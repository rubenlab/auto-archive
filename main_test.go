package main

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	err := loadConfig("./config-test.yml")
	if err != nil {
		t.Error(err)
	}
	failed := appConfig.DB != "archive.db"
	failed = failed || appConfig.ScanLevel != 3
	failed = failed || appConfig.ArchiveInterval != 30
	failed = failed || appConfig.ArchiveCommand != "rm -rf \"${path}\""
	failed = failed || appConfig.LogFolder != "log"
	failed = failed || appConfig.Cores != 4
	if failed {
		t.Errorf("config value not correct: \n%v", appConfig)
	}
}

func TestDoArchive(t *testing.T) {
	err := execArchiveCommand("./config-test.yml", "test", "echo \"${path}\"")
	if err != nil {
		t.Error(err)
	}
}

// func TestSendNotice(t *testing.T) {
// 	var scanResult ScanResult = ScanResult{
// 		Errors: []ScanError{
// 			{Path: "/test1", Msg: "err1"},
// 			{Path: "/test2", Msg: "err2"},
// 		},
// 		Notices: []ArchiveNotice{
// 			{
// 				Path:              "/test3",
// 				DaysBeforeArchive: 10,
// 			},
// 			{
// 				Path:              "/test4",
// 				DaysBeforeArchive: 5,
// 			},
// 		},
// 		ArchivedFolders: []ArchivedFolder{
// 			{
// 				Path: "/test5",
// 			},
// 			{
// 				Path: "/test6",
// 			},
// 		},
// 	}
// 	err := sendNoticeInternal(&scanResult, &EmailConfig{
// 		ServerName: "test server",
// 		Host:       "smtp-mail.outlook.com",
// 		Port:       587,
// 		From:       "rubsak1@outlook.com",
// 		To:         "tianming.yi@med.uni-goettingen.de",
// 		User:       "rubsak1@outlook.com",
// 		Password:   "efndkubpkaksfmsx",
// 	})
// 	if err != nil {
// 		t.Error(err)
// 	}
// }
