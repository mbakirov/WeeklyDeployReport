package main

import (
	"fmt"
	"os"
	"time"
	"github.com/joho/godotenv"
)

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

	xlsFile := &CalendarTable{
		header: []XlsHeaderCol{
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
		},
		startDate: weekStart,
		endDate: weekEnd,
		path: fmt.Sprintf("cache/Календарь_внедрений_СБЕР_ЕАПТЕКА_%s-%s.xlsx", weekStart.Format("02.01"), weekEnd.Format("02.01")),
		headerStyle: `{
			"border": [{"type":"1", "style": 1, "color":"#000000"}],
			"fill": {"type":"pattern", "color":["#90C225"], "pattern":1},
			"font": {"family":"Calibri", "size":8.0, "bold":true, "color":"#FFFFFF"},
			"alignment": {"wrap_text":true, "horizontal":"center", "vertical":"center"}
		}`,
		bodyStyle: `{
			"border": [{"type":"1", "style":1, "color": "#000000"}],
			"font": {"family":"Calibri", "size":10.0, "color":"#000000"},
			"alignment": {"wrap_text":false, "horizontal":"left", "vertical":"top"}
		}`,
		mailSubject: fmt.Sprintf("Календарь внедрений СБЕР ЕАПТЕКА %s-%s", weekStart.Format("02.01"), weekEnd.Format("02.01")),
		mailBody: "Коллеги, добрый день!\n\nКалендарь внедрений за эту неделю во вложении.",
		//debugMode: true,
	}

	xlsFile.CreateFile()

	issues := VersionsIssues{
		startDate: weekStart,
		endDate: weekEnd,
		releaseDateFormat: "2006-01-02",
		releaseDateRegex: `^\d{4}-\d{2}-\d{2}$`,
	}

	for _, issue := range issues.GetIssues() {
		xlsFile.AddRow([]string{
			issue.GetKey(),
			issue.GetSummary(),
			issue.GetServiceName(),
			issue.GetUnavailability(),
			issue.GetDeployStart(),
			issue.GetDeployEnd(),
			issue.GetDeployStatus(),
			issue.GetDeployManager(),
			issue.GetMaintainManager(),
			issue.GetDeployRisk(),
			issue.GetDeployResult(),
		})
	}

	xlsFile.Write()

	xlsFile.Send()

	os.Exit(0)
}