package main

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
