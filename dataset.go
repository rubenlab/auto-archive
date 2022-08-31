package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const DatasetFileName = ".datasetinfo"
const FramesFolderName = "frames"

var CharacterFolderNames = [...]string{"frames", "Images-Disc1"}

type Datasetinfo struct {
	ID         string       `yaml:"id"`
	BackupTime sql.NullTime `yaml:"backup-time"`
}

// read dataset info stored in the .datasetinfo file of a dataset folder
// path: path to the dataset folder
func ReadDatasetinfo(path string) (*Datasetinfo, error) {
	datasetfilePath := filepath.Join(path, DatasetFileName)
	data, err := ioutil.ReadFile(datasetfilePath)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("can not open config file %s", path))
	}
	info := Datasetinfo{}
	err = yaml.Unmarshal(data, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// If the path is a valid folder, check if it's a dataset folder by the following rules:
//
// 1. If a .datasetinfo file is under this folder
//
// 2. If a folder named "frames" is under this folder
//
// If it's a dataset folder, but there's no .datasetinfo file inside it, create it and add the record to the database
//
// If there's a .datasetinfo file inside, but the id is not recorded by the database, add it to the database
//
// return true if it's a dataset folder
func CreateIfDataset(path string) (bool, error) {
	datasetfilePath := filepath.Join(path, DatasetFileName)
	_, err := os.Stat(datasetfilePath)
	if err != nil { // .datasetinfo folder doesn't exist
		if containsCharacterFolder(path) {
			err = AddDataset(path)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	} else { // if .datasetinfo folder already exists, then it's a dataset folder
		data, err := ioutil.ReadFile(datasetfilePath)
		if err != nil {
			return false, err
		}
		info := Datasetinfo{}
		err = yaml.Unmarshal(data, &info)
		if err != nil {
			return false, err
		}
		id := info.ID
		record, err := GetRecord(id)
		if err != nil {
			return false, err
		}
		if record == nil {
			record = &DatasetRecord{
				ID:   id,
				Path: path,
			}
			UpdateRecord(record)
		} else if record.Path != path {
			record.Path = path
			UpdateRecord(record)
		}
		return true, nil
	}
	return false, nil
}

// contains character folder that can decide it's a dataset folder
func containsCharacterFolder(path string) bool {
	for _, dir := range CharacterFolderNames {
		dirPath := filepath.Join(path, dir)
		d, err := os.Stat(dirPath)
		if err == nil && d.IsDir() { //frames folder exists
			return true
		}
	}
	return false
}

// Add a folder as a dataset folder, it will do the following things:
//
// 1. add a .datasetinfo file to the folder
//
// 2. add the record to the database
func AddDataset(path string) error {
	id, err := createDatasetInfo(path)
	if err != nil {
		return err
	}
	record := DatasetRecord{
		ID:   id,
		Path: path,
	}
	err = AddRecord(&record)
	return err
}

// Create a .datasetinfo file to the folder
// add id property inside it
//
// path is the path of folder to be marked as dataset
//
// return it's id
func createDatasetInfo(path string) (string, error) {
	datasetfilePath := filepath.Join(path, DatasetFileName)
	id := uuid.New().String()
	info := Datasetinfo{ID: id}
	data, err := yaml.Marshal(info)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(datasetfilePath, data, os.FileMode(int(0640)))
	if err != nil {
		return "", err
	}
	return id, nil
}

func SaveDatasetInfo(path string, info *Datasetinfo) error {
	datasetfilePath := filepath.Join(path, DatasetFileName)
	data, err := yaml.Marshal(info)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(datasetfilePath, data, os.FileMode(int(0640)))
	return err
}
