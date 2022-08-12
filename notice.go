package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net/smtp"
	"text/template"
	"time"

	"github.com/jordan-wright/email"
)

const tpl = `
<h1>Directories to be archived</h1>
{{range .Notices}}<p>{{ .Path }} will be archived in {{ .DaysBeforeArchive }} days</p>{{end}}
<h1>Directories archived today</h1>
{{range .ArchivedFolders}}<p>{{ .Path }}</p>{{end}}
<h1>Errors</h1>
{{range .Errors}}<p>folder: {{ .Path }} error: {{ .Msg }}</p>{{end}}
`

func SendNotice(scanResult *ScanResult) error {
	config := EmailConfig{
		ServerName: appConfig.ServerName,
		Host:       appConfig.SmtpHost,
		Port:       appConfig.SmtpPort,
		From:       appConfig.SmtpUser,
		To:         appConfig.EmailTo,
		User:       appConfig.SmtpUser,
		Password:   appConfig.SmtpPassword,
	}
	return sendNoticeInternal(scanResult, &config)
}

type EmailConfig struct {
	ServerName string
	Host       string
	Port       int
	From       string
	To         string
	User       string
	Password   string
}

func sendNoticeInternal(scanResult *ScanResult, c *EmailConfig) error {
	// if nothing to notice, return
	if len(scanResult.Errors) == 0 && len(scanResult.ArchivedFolders) == 0 && len(scanResult.Notices) == 0 {
		return nil
	}
	t, err := template.New("emailbody").Parse(tpl)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	out := bufio.NewWriter(&buf)
	err = t.Execute(out, scanResult)
	out.Flush()
	if err != nil {
		return err
	}
	e := email.NewEmail()
	e.From = c.From
	e.To = []string{c.To}
	now := time.Now()
	timeStr := now.Format("Mon, 02 Jan 2006")
	e.Subject = fmt.Sprintf("Archive report %s %s", c.ServerName, timeStr)
	e.HTML = buf.Bytes()
	var auth smtp.Auth
	if c.Host != "localhost" {
		if c.Host == "smtp-mail.outlook.com" {
			auth = LoginAuth(c.User, c.Password)
		} else {
			auth = smtp.PlainAuth("", c.User, c.Password, c.Host)
		}
	}
	err = e.Send(fmt.Sprintf("%s:%d", c.Host, c.Port), auth)
	return err
}
