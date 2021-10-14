package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"context"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/andygrunwald/go-jira"
	"github.com/mailgun/mailgun-go"
	"github.com/joho/godotenv"
)

// jira api doc https://developer.atlassian.com/cloud/jira/platform/rest/v2/api-group-project-versions/
// go-jira api doc https://pkg.go.dev/net/http#Response
// excelize https://medium.com/cloud-native-the-gathering/using-golang-to-create-and-read-excel-files-7e0c10a31583

const releaseDateFormat = "2006-01-02"

// init is invoked before main()
func init() {
    // loads values from .env into the system
    if err := godotenv.Load(); err != nil {
        fmt.Println("No .env file found")
		os.Exit(255)
    }
}

func main() {
	weekEnd := time.Now()
	weekStart := weekEnd.AddDate(0, 0, -7)
	weekStart.AddDate(0, 0, 1)
	
	issues := getIssues(weekStart, weekEnd)

	xls := excelize.NewFile()
	xls.SetActiveSheet(1)

	xlsHeader := []xlsCol{
		{col:"A", name:"Код / ID", width:15},
		{col:"B", name:"Описание", width:50},
		{col:"C", name:"Сервис", width:10},
		{col:"D", name:"Влияние на ключевые бизнес-процесы", width:25},
		{col:"E", name:"Дата время начала", width:15},
		{col:"F", name:"Дата время окончания работ", width:15},
		{col:"G", name:"Статус внедрения"},
		{col:"H", name:"Менеджер внедрения"},
		{col:"I", name:"Менеджер сопровождения"},
		{col:"J", name:"Риски внедрения"},
		{col:"K", name:"Результат внедрения"},
	}

	makeHeaderRow(xls, xlsHeader)

	makeBodyRow(xls, xlsHeader, issues)

	// Save spreadsheet by the given path.
	if err := xls.SaveAs("cache/Календарь внедрений.xlsx"); err != nil {
		fmt.Println(err)
	}

	sendFile(xls, weekStart, weekEnd)

	os.Exit(0)
}

func parseServiceName(issue jira.Issue) string {
	issueVersion := issue.Fields.FixVersions[len(issue.Fields.FixVersions)-1]

	re := regexp.MustCompile(`^([A-Za-z_\-]+)(\s.+)$`)
	return re.ReplaceAllString(issueVersion.Name, "$1")
}

func parseDeployRisk(issue jira.Issue) string {
	prioriryMap := make(map[string]string)

	prioriryMap["Critical"] = "Высокий"
	prioriryMap["High"] = "Высокий"
	prioriryMap["Medium"] = "Средний"
	prioriryMap["Low"] = "Низкий"

	return prioriryMap[issue.Fields.Priority.Name]
}

func parseManagerDeploy(issue jira.Issue) string {
	re := regexp.MustCompile(`(\s\(.+\))$`)
	assignee := issue.Fields.Assignee.DisplayName
	manager := issue.Fields.Unknowns["customfield_12507"]

	if manager != nil {
		assignee = manager.(map[string]interface{})["displayName"].(string)
	}

	return strings.Trim(re.ReplaceAllString(assignee, ""), " ()")
}

func parseDeployDatetime(issue jira.Issue) (string, string) {
	defaultTime := map[string]string{"start":"12:00","end":"14:00"}

	serviceReleaseTime := map[string]map[string]string{
		"ERP": {"start": "23:00", "end": "03:00"},
		"WMS": {"start": "16:00", "end": "17:00"},
		"MDLP": {"start": "12:00", "end": "13:00"},
		"WEB": {"start": "13:00", "end": "15:00"},
		"IOS": {"start": "10:00", "end": "11:00"},
		"ANDR": {"start": "10:00", "end": "11:00"}}

	serviceName := parseServiceName(issue)

	serviceTime, ok := serviceReleaseTime[serviceName]
	if !ok {
		serviceTime = defaultTime
	}

	issueVersion := issue.Fields.FixVersions[len(issue.Fields.FixVersions)-1]
	releaseStartDate, _ := time.Parse(releaseDateFormat, issueVersion.ReleaseDate)
	releaseEndDate := releaseStartDate

	if serviceName == "ERP" {
		releaseEndDate = releaseStartDate.AddDate(0, 0, 1)
	}

	return fmt.Sprintf("%s %s", releaseStartDate.Format("2006-01-02"), serviceTime["start"]),
		fmt.Sprintf("%s %s", releaseEndDate.Format("2006-01-02"), serviceTime["end"])
}

func makeHeaderRow(xls *excelize.File, xlsHeader []xlsCol) {
	fmt.Println("Creating header line...")

	for _, header := range xlsHeader {
		err := xls.SetCellValue("Sheet1", header.col + "1", header.name)
		if err != nil {
			fmt.Printf("Error while setting header value: %s\n", err)
			os.Exit(2)
		}
	}

	fmt.Println("Creating header style...")
	xlsHeaderStyle, err := xls.NewStyle(`{
		"border": [{"type":"1", "style": 1, "color":"#000000"}],
		"fill": {"type":"pattern", "color":["#90C225"], "pattern":1},
		"font": {"family":"Calibri", "size":8.0, "bold":true, "color":"#FFFFFF"},
		"alignment": {"wrap_text":true, "horizontal":"center", "vertical":"center"}
	}`)

	if err != nil {
		fmt.Printf("Error while creating header style: %s\n", err)
		os.Exit(2)
	}

	fmt.Println("Applying header style...")
	err = xls.SetCellStyle("Sheet1", fmt.Sprintf("%s%d", xlsHeader[0].col, 1), fmt.Sprintf("%s%d", xlsHeader[len(xlsHeader)-1].col, 1), xlsHeaderStyle)

	if err != nil {
		fmt.Printf("Error while applying body style: %s\n", err)
		os.Exit(2)
	}
}

func makeBodyRow(xls *excelize.File, xlsHeader []xlsCol, issues []jira.Issue) {
	xlsRow := []string{}

	rowIndex := 2
	for _, issue := range issues {
		re := regexp.MustCompile(`(\s\(.+\))$`)
		assignee := re.ReplaceAllString(issue.Fields.Assignee.DisplayName, "")

		startTime, endTime := parseDeployDatetime(issue)

		xlsRow = []string{
			issue.Key,
			issue.Fields.Summary,
			parseServiceName(issue),
			"Недоступность ключевых бизнес-сервсов не пларинуется",
			startTime,
			endTime,
			"Выполнено",
			parseManagerDeploy(issue),
			strings.Trim(assignee, " ()"),
			parseDeployRisk(issue),
			"успешно",
		}

		for i, r := range xlsRow {
			err := xls.SetCellValue("Sheet1", fmt.Sprintf("%s%d", xlsHeader[i].col, rowIndex), r)

			if err != nil {
				fmt.Printf("Error while setting cell value: %s\n", err)
				os.Exit(2)
			}
		}

		rowIndex++
	}

	xlsBodyStyle, err := xls.NewStyle(`{
		"border": [{"type":"1", "style":1, "color": "#000000"}],
		"font": {"family":"Calibri", "size":10.0, "color":"#000000"},
		"alignment": {"wrap_text":false, "horizontal":"left", "vertical":"top"}
	}`)

	for _, row := range xlsHeader {
		err := xls.SetColWidth("Sheet1", row.col, row.col, row.width)
		
		if err != nil {
			fmt.Printf("Error while setting col width: %s\n", err)
			os.Exit(2)
		}
	}

	if err != nil {
		fmt.Printf("xlsBodyStyle error - %s\n", err)
		os.Exit(2)
	}

	err = xls.SetCellStyle("Sheet1", "A2", fmt.Sprintf("%s%d", xlsHeader[len(xlsHeader)-1].col, rowIndex), xlsBodyStyle)

	if err != nil {
		fmt.Printf("xlsHeaderStyle set error - %s\n", err)
		os.Exit(2)
	}

	fmt.Printf("Formating body: sheet - %s, start - %s, end - %s\n", "Sheet1", "A2", fmt.Sprintf("%s%d", xlsHeader[len(xlsHeader)-1].col, rowIndex))
}

func getIssues(weekStart time.Time, weekEnd time.Time) []jira.Issue {
	login, _ := os.LookupEnv("JIRA_LOGIN")
	password, _ := os.LookupEnv("JIRA_PASSWORD")
	url, _ := os.LookupEnv("JIRA_URL")

	tp := jira.BasicAuthTransport{
		Username: login,
		Password: password,
	}

	jiraClient, _ := jira.NewClient(tp.Client(), url)

	fmt.Printf("Requesting projects: from %s as %s\n", url, login)
	projectList, _, _ := jiraClient.Project.GetList()

	releasedVersions := []string{}

	fmt.Printf("INFO: requesting versions dates between dates: %#v-%#v\n", weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))

	for _, project := range *projectList {
		if (project.ProjectTypeKey != "software") {
			continue
		}

		fmt.Printf("\rINFO: requesting versions, project - %s...", project.Key)
		request, err := jiraClient.NewRequest("GET", "/rest/api/2/project/"+project.Key+"/version?maxResults=5000&status=released", nil)

		if (err != nil) {
			fmt.Printf("\nERROR: requesting versions, project - %s, error - %s...\n", project.Key, err)
		}

		versionsList := new(VersionList)
		_, _ = jiraClient.Do(request, versionsList)

		counter := 0
		for _, version := range versionsList.Values {
			matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, version.ReleaseDate)

			if !matched {
				continue
			}

			releaseDate, _ := time.Parse(releaseDateFormat, version.ReleaseDate)

			if releaseDate.Before(weekStart) || releaseDate.After(weekEnd) {
				continue
			}

			releasedVersions = append(releasedVersions, version.ID)
			counter++
		}

		if (counter > 0) {
			fmt.Printf("\rINFO: obtained %d versions, project - %s...\n", counter, project.Key)
		}
	}

	opt := &jira.SearchOptions{
		MaxResults: 2000,
	}

	jql := fmt.Sprintf("fixVersion in (%s) and labels not in (RELEASEBUG) order by fixVersion ASC", strings.Join(releasedVersions, ","))

	issues := []jira.Issue{}

	fmt.Printf("\nINFO: requesting issuess: %s...\n", jql)
	chunk, resp, err := jiraClient.Issue.Search(jql, opt)

	if err != nil {
		fmt.Printf("ERROR: error while issues search: %s...\n", err)
		return issues
	}

	fmt.Printf("INFO: found %d issues\n", resp.Total)

	return append(issues, chunk...)
}

func sendFile(xls *excelize.File, weekStart time.Time, weekEnd time.Time) {
	domain, _ := os.LookupEnv("MAILGUN_DOMAIN")
	key, _ := os.LookupEnv("MAILGUN_DOMAIN")
	sender, _ := os.LookupEnv("EMAIL_SENDER")
	recipients, _ := os.LookupEnv("RECIPIENTS")

	mg := mailgun.NewMailgun(domain, key)
	subject := fmt.Sprintf("Календарь внедрений СБЕР ЕАПТЕКА %s-%s", weekStart.Format("02.01"), weekEnd.Format("02.01"))
	body := "Коллеги, добрый день!\n\nКалендарь внедрений за эту неделю во вложении."

	message := mg.NewMessage(sender, subject, body, sender)
	message.AddBCC(recipients)

	fmt.Printf("INFO: mailgun xls path: %s\n", xls.Path)

	message.AddAttachment(xls.Path)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err := mg.Send(ctx, message)

	if err != nil {
		fmt.Printf("ERROR: mailgun error:\n%s\n", err)
		os.Exit(1)
	}

	fmt.Printf("INFO: mailgun sended to: %s\n",  recipients)
}

type xlsCol struct{
	col string
	name string
	width float64
}