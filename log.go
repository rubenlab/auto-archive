package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var logOutputFolder string

const logFileName = "autoarchive.log"

func initLog() (io.Closer, error) {
	logFolder := appConfig.LogFolder
	if logFolder == "" {
		log.Println("warning: LogFolder is empty, logs will be written to current working directory")
	}
	logStartTime := time.Now()
	dateStr := logStartTime.Format("2006-01-02")
	logOutputFolder = filepath.Join(logFolder, dateStr)
	os.MkdirAll(logOutputFolder, FolderModeCreate)
	logFile := filepath.Join(logOutputFolder, logFileName)
	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE, FileModeCreate)
	if err != nil {
		return nil, err
	}
	log.Printf("write log output to file: %s\n", logFile)
	log.SetOutput(file)
	return file, err
}
