package main

import (
	"fmt"
	"os"
	"time"
	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/mailgun/mailgun-go"
)

type XlsHeaderCol struct {
	col   string
	name  string
	width float64
}

type CalendarTable struct {
	header       []XlsHeaderCol
	excelize     *excelize.File
	lastRowIndex int
	path         string
	startDate    time.Time
	endDate      time.Time
	headerStyle  string
	bodyStyle    string
	mailSubject  string
	mailBody     string
	debugMode    bool
}

func (file *CalendarTable) CreateFile() *CalendarTable {
	if file.header == nil {
		fmt.Println("ERROR: Table header is empty")
		os.Exit(2)
	}

	file.excelize = excelize.NewFile()
	file.lastRowIndex = 1

	header := []string{}
	for _, r := range file.header {
		header = append(header, r.name)
	}

	file.AddRow(header)
	file.formatHeaderStyle()

	return file
}

func (file *CalendarTable) formatHeaderStyle() {
	fmt.Println("INFO: Creating header line...")

	for _, header := range file.header {
		file.excelize.SetCellValue("Sheet1", header.col+"1", header.name)
	}

	xlsHeaderStyle, err := file.excelize.NewStyle(file.headerStyle)

	if err != nil {
		fmt.Printf("ERROR: while creating header style: %s\n", err)
		os.Exit(2)
	}

	fmt.Println("INFO: Applying header style...")
	file.excelize.SetCellStyle("Sheet1", fmt.Sprintf("%s%d", file.header[0].col, 1), fmt.Sprintf("%s%d", file.header[len(file.header)-1].col, 1), xlsHeaderStyle)

	for _, row := range file.header {
		file.excelize.SetColWidth("Sheet1", row.col, row.col, row.width)
	}
}

func (file *CalendarTable) formatBodyStyle() {
	xlsBodyStyle, err := file.excelize.NewStyle(file.bodyStyle)

	if err != nil {
		fmt.Printf("ERROR: while creating header style: %s\n", err)
		os.Exit(2)
	}

	file.excelize.SetCellStyle("Sheet1", "A2", fmt.Sprintf("%s%d", file.header[len(file.header)-1].col, file.lastRowIndex), xlsBodyStyle)

	fmt.Printf("INFO: Formating body, sheet - %s, start - %s, end - %s\n", "Sheet1", "A2", fmt.Sprintf("%s%d", file.header[len(file.header)-1].col, file.lastRowIndex))
}

func (file *CalendarTable) Write() {
	file.formatBodyStyle()

	err := file.excelize.SaveAs(file.path)

	if err != nil {
		fmt.Printf("ERROR: while writing file: %s\n", err)
		os.Exit(2)
	}

	fmt.Printf("INFO: file saved: %#v\n", file.path)
}

func (file *CalendarTable) Send() {
	domain, _ := os.LookupEnv("MAILGUN_DOMAIN")
	key, _ := os.LookupEnv("MAILGUN_KEY")
	sender, _ := os.LookupEnv("EMAIL_SENDER")
	recipients, _ := os.LookupEnv("RECIPIENTS")

	mg := mailgun.NewMailgun(domain, key, "")
	subject := file.mailSubject
	body := file.mailBody

	if file.debugMode {
		fmt.Println("INFO: Email send prevented due DEBUG MODE")
		os.Exit(100)
	}

	message := mg.NewMessage(sender, subject, body, sender)
	message.AddBCC(recipients)
	message.AddAttachment(file.path)

	mess, id, err := mg.Send(message)

	if err != nil {
		fmt.Printf("ERROR: mailgun error: %s\nmessage:%s\nid:%s\n", err, mess, id)
		os.Exit(1)
	}

	fmt.Printf("INFO: mailgun sended to: %s\n", recipients)
}

func (file *CalendarTable) AddRow(rowData []string) {
	if file.debugMode {
		fmt.Printf("INFO: Creating %d row...\n", file.lastRowIndex)
	}

	for i, r := range rowData {
		file.excelize.SetCellValue("Sheet1", fmt.Sprintf("%s%d", file.header[i].col, file.lastRowIndex), r)
	}

	file.lastRowIndex++
}
