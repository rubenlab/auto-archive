package main

import (
	"bytes"
	"encoding/csv"
	"log"
	"os/exec"
	"strings"
)

func doArchive(path string, id string) error {
	archiveCommand := appConfig.ArchiveCommand
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
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Printf("output of archive command '%s':\n%s", archiveCommand, out.String())
	return nil
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
