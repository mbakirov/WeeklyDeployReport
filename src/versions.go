package main

import (
	//"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
)

type VersionList struct {
	Self       string    `json:"self"`
	NextPage   string    `json:"nextPage"`
	MaxResults int       `json:"maxResults"`
	StartAt    int       `json:"startAt"`
	Total      int       `json:"total"`
	IsLast     bool      `json:"isLast"`
	Values     []Version `json:"values"`
}

type Version struct {
	Self            string `json:"self"`
	ID              string `json:"id"`
	Description     string `json:"description,omitempty"`
	Name            string `json:"name"`
	Archived        bool   `json:"archived"`
	Released        bool   `json:"released"`
	ReleaseDate     string `json:"releaseDate"`
	UserReleaseDate string `json:"userReleaseDate"`
	ProjectID       int    `json:"projectId"`
	Overdue         bool   `json:"overdue,omitempty"`
}

type VersionsIssues struct {
	jiraClient        *jira.Client
	releaseDateFormat string
	releaseDateRegex  string
	startDate         time.Time
	endDate           time.Time
	issues            []jira.Issue
	skipProjects      string
}

func (this *VersionsIssues) getClient() *jira.Client {
	if this.jiraClient != nil {
		return this.jiraClient
	}

	login, _ := os.LookupEnv("JIRA_LOGIN")
	password, _ := os.LookupEnv("JIRA_PASSWORD")
	url, _ := os.LookupEnv("JIRA_URL")

	tp := jira.BasicAuthTransport{
		Username: login,
		Password: password,
	}

	jiraClient, err := jira.NewClient(tp.Client(), url)

	if jiraClient == nil && err != nil {
		fmt.Printf("ERROR: Couldn't connect to jira: %s\n", err)
		os.Exit(2)
	}

	jiraClient.Project.GetList()

	fmt.Printf("INFO: Connected to jira server: %s with login %s\n", url, login)

	this.jiraClient = jiraClient

	return this.jiraClient
}

var skipProjectWasRead bool
var skipProjects []string

func (this *VersionsIssues) isProjectSkipped(key string) bool {
	result := false

	if !skipProjectWasRead {
		skipProjects = strings.Split(this.skipProjects, ",")
	}

	for _, skipKey := range skipProjects {
		if key == skipKey {
			result = true
		}
	}

	return result
}

func (this *VersionsIssues) getProjects() jira.ProjectList {
	fmt.Println("INFO: Requesting projects...")

	projectList, _, err := this.getClient().Project.GetList()

	if err != nil {
		fmt.Printf("ERROR: Couldn't get projects: %s\n", err)
		os.Exit(2)
	}

	var result jira.ProjectList

	for _, project := range *projectList {
		if project.ProjectTypeKey != "software" || this.isProjectSkipped(project.Key) {
			fmt.Printf("INFO: Skipping project: %s\n", project.Key)
			continue
		}

		result = append(result, project)
	}

	fmt.Printf("INFO: obtained %d projects\n", len(result))

	return result
}

func (this *VersionsIssues) getProjectVersions(projectKey string) []string {
	//fmt.Printf("\rINFO: requesting versions, project - %s...", projectKey)

	request, err := this.getClient().NewRequest("GET", "/rest/api/2/project/"+projectKey+"/version?maxResults=5000&status=released", nil)

	if err != nil {
		fmt.Printf("\nERROR: requesting versions for project %s, error - %s...\n", projectKey, err)
		os.Exit(2)
	}

	var releasedVersions []string
	versionsList := new(VersionList)
	_, err = this.getClient().Do(request, versionsList)

	if err != nil {
		fmt.Printf("\nERROR: Couldn't get project version: %s\n", err)
		os.Exit(2)
	}

	for _, version := range versionsList.Values {
		matched, _ := regexp.MatchString(this.releaseDateRegex, version.ReleaseDate)

		if !matched {
			//fmt.Printf("\nINFO: skipping version %s (%s) - date not match...", version.ID, projectKey)
			continue
		}

		releaseDate, _ := time.Parse(this.releaseDateFormat, version.ReleaseDate)

		if releaseDate.Before(this.startDate) || releaseDate.After(this.endDate) {
			//fmt.Printf("\nINFO: skipping version %s (%s) - release date not match... ", version.ID, projectKey)
			continue
		}

		releasedVersions = append(releasedVersions, version.ID)
	}

	return releasedVersions
}

func (this *VersionsIssues) getVersions() []string {
	releasedVersions := []string{}
	projectList := this.getProjects()

	fmt.Printf("INFO: requesting versions...\n")

	for _, project := range projectList {
		releasedVersions = append(releasedVersions, this.getProjectVersions(project.Key)...)
	}

	if len(releasedVersions) > 0 {
		fmt.Printf("INFO: obtained %d versions\n", len(releasedVersions))
	}

	return releasedVersions
}

func (this *VersionsIssues) GetIssues() []Issue {
	releasedVersions := this.getVersions()

	jql := fmt.Sprintf("fixVersion in (%s) and (labels is EMPTY or labels not in (RELEASEBUG)) order by fixVersion ASC", strings.Join(releasedVersions, ","))
	fmt.Printf("INFO: requesting issuess with JQL: %s...\n", jql)

	opt := &jira.SearchOptions{
		MaxResults: 2000,
	}

	chunk, resp, err := this.getClient().Issue.Search(jql, opt)

	if err != nil {
		fmt.Printf("ERROR: error while issues search: %s...\n", err)
		os.Exit(2)
	}

	fmt.Printf("INFO: found %d issues\n", resp.Total)

	result := []Issue{}

	for _, item := range chunk {
		result = append(result, Issue{
			jira:              item,
			projectRegex:      `^([A-Za-z_\-]+)(\s.+)$`,
			releaseDateFormat: "2006-01-02",
		})
	}

	return result
}

type Issue struct {
	jira              jira.Issue
	projectRegex      string
	releaseDateFormat string
	count             int
	deployEnd         string
	deployStart       string
}

func (issue *Issue) GetKey() string {
	return issue.jira.Key
}

func (issue *Issue) GetSummary() string {
	return issue.jira.Fields.Summary
}

func (issue *Issue) GetServiceName() string {
	issueVersion := issue.jira.Fields.FixVersions[len(issue.jira.Fields.FixVersions)-1]

	re := regexp.MustCompile(issue.projectRegex)

	return re.ReplaceAllString(issueVersion.Name, "$1")
}

func (issue *Issue) GetUnavailability() string {
	return "Недоступность ключевых бизнес-сервсов не пларинуется"
}

func (issue *Issue) parseDeployDatetime() {
	if issue.deployStart != "" && issue.deployEnd != "" {
		return
	}

	defaultTime := map[string]string{"start": "12:00", "end": "14:00"}

	serviceReleaseTime := map[string]map[string]string{
		"ERP":  {"start": "23:00", "end": "03:00"},
		"WMS":  {"start": "16:00", "end": "17:00"},
		"MDLP": {"start": "12:00", "end": "13:00"},
		"WEB":  {"start": "13:00", "end": "15:00"},
		"IOS":  {"start": "10:00", "end": "11:00"},
		"ANDR": {"start": "10:00", "end": "11:00"}}

	serviceName := issue.GetServiceName()

	serviceTime, ok := serviceReleaseTime[serviceName]
	if !ok {
		serviceTime = defaultTime
	}

	issueVersion := issue.jira.Fields.FixVersions[len(issue.jira.Fields.FixVersions)-1]
	releaseStartDate, _ := time.Parse(issue.releaseDateFormat, issueVersion.ReleaseDate)
	releaseEndDate := releaseStartDate

	if serviceName == "ERP" {
		releaseEndDate = releaseStartDate.AddDate(0, 0, 1)
	}

	issue.deployStart = fmt.Sprintf("%s %s", releaseStartDate.Format("2006-01-02"), serviceTime["start"])
	issue.deployEnd = fmt.Sprintf("%s %s", releaseEndDate.Format("2006-01-02"), serviceTime["end"])
}

func (issue *Issue) GetDeployStart() string {
	issue.parseDeployDatetime()
	return issue.deployStart
}

func (issue *Issue) GetDeployEnd() string {
	issue.parseDeployDatetime()
	return issue.deployEnd
}

func (issue *Issue) GetDeployStatus() string {
	return "Выполнено"
}

func (issue *Issue) GetDeployManager() string {
	re := regexp.MustCompile(`(\s\(.+\))$`)
	assignee := issue.getAssigneeName()
	manager := issue.jira.Fields.Unknowns["customfield_12507"]

	if manager != nil {
		assignee = manager.(map[string]interface{})["displayName"].(string)
	}

	return strings.Trim(re.ReplaceAllString(assignee, ""), " ()")
}

func (issue *Issue) GetMaintainManager() string {
	return strings.Trim(issue.getAssigneeName(), " ()")
}

func (issue *Issue) GetDeployRisk() string {
	prioriryMap := map[string]string{
		"Critical": "Высокий",
		"High":     "Высокий",
		"Medium":   "Средний",
		"Low":      "Низкий",
	}

	return prioriryMap[issue.jira.Fields.Priority.Name]
}

func (issue *Issue) getAssigneeName() string {
	assignee := ""

	if issue.jira.Fields.Assignee != nil {
		assignee = issue.jira.Fields.Assignee.DisplayName
	} else if issue.jira.Fields.Reporter != nil {
		assignee = issue.jira.Fields.Reporter.DisplayName
	}

	return assignee
}

func (issue *Issue) GetDeployResult() string {
	return "успешно"
}
