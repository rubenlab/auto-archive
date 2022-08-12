package main

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type AppConfig struct {
	DB              string `yaml:"db"`
	ServerName      string `yaml:"server-name"` // server name for email report
	Root            string // root path to scan
	ScanLevel       int    `yaml:"scan-level"`       // If the scan depth reaches ScanLevel, force the directories to be marked as dataset
	ScanInterval    int    `yaml:"scan-interval"`    // scan interval in days
	ArchiveInterval int    `yaml:"archive-interval"` // archive interval in days
	NoticeBefore    []int  `yaml:"notice-before"`    // how many days to notice before archive
	EmailTo         string `yaml:"email-to"`         // email to whom when folder will be archived
	ArchiveCommand  string `yaml:"archive-command"`  // archive command, ${path} can be used.
	SmtpHost        string `yaml:"smtp-host"`        // smtp host address
	SmtpPort        int    `yaml:"smtp-port"`        // smtp port
	SmtpUser        string `yaml:"smtp-user"`        // smtp username
	SmtpPassword    string `yaml:"smtp-password"`    // smtp password
}

var appConfig *AppConfig = &AppConfig{
	DB:              "archive.db",
	ScanLevel:       3,
	ScanInterval:    3,
	ArchiveInterval: 30,
	NoticeBefore:    []int{10, 5, 1},
	SmtpHost:        "localhost", // will use local email server
}

func loadConfig(path string) error {
	if path == "" {
		return nil
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, "can not open config file")
	}
	config := AppConfig{}
	err = yaml.Unmarshal(data, &config)
	appConfig = &config
	if err != nil {
		return errors.Wrap(err, "can not unmarshal config data")
	}
	if appConfig.ArchiveCommand == "" {
		return errors.New("archive command must be provided")
	}
	return nil
}