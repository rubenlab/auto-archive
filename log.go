package main

import (
	"log"
	"os"
)

func initLog() error {
	logFile := appConfig.LogFile
	if logFile == "" {
		return nil
	}
	file, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	log.SetOutput(file)
	return nil
}
