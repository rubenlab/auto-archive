package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func doArchive(path string, id string) error {
	archiveCommand := appConfig.ArchiveCommand
	if archiveCommand == "" {
		return errors.New("archive command is empty, this folder should be archived")
	}
	return execArchiveCommand(path, id, archiveCommand)
}

func execArchiveCommand(path string, id string, archiveCommand string) error {
	archiveCommand = strings.Replace(archiveCommand, "${id}", id, -1)
	archiveCommand = strings.Replace(archiveCommand, "${path}", path, -1)
	fields, err := getFields(archiveCommand)
	if err != nil {
		return err
	}
	name := fields[0]
	args := fields[1:]
	cmd := exec.Command(name, args...)
	rc, logErr := getArchiveWriter(path, id)
	defer func() {
		if rc != nil {
			rc.Close()
		}
	}()
	if logErr == nil {
		cmd.Stdout = rc
	}
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func getLogWriter(fileName string, title string) (io.WriteCloser, error) {
	logFilePath := filepath.Join(logOutputFolder, fileName)
	file, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE, FileModeCreate)
	if err != nil {
		return nil, err
	}
	file.WriteString(title)
	return file, nil
}

func getArchiveWriter(path string, id string) (io.WriteCloser, error) {
	logFileName := "archive_" + id + ".log"
	title := fmt.Sprintf("folder path: %s\n", path)
	return getLogWriter(logFileName, title)
}

func getFields(str string) ([]string, error) {
	r := csv.NewReader(strings.NewReader(str))
	r.Comma = ' ' // space
	fields, err := r.Read()
	if err != nil {
		return nil, err
	}
	return fields, nil
}
