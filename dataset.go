package main

import (
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

type Datasetinfo struct {
	ID string `yaml:"id"`
}

func readDatasetinfo(path string) (*Datasetinfo, error) {
	data, err := ioutil.ReadFile(path)
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
	datasetfilePath := filepath.Join(path, ".datasetinfo")
	_, err := os.Stat(datasetfilePath)
	if err != nil { // .datasetinfo folder doesn't exist
		framesFolderPath := filepath.Join(path, FramesFolderName)
		d, err := os.Stat(framesFolderPath)
		if err == nil && d.IsDir() { //frames folder exists
			err = AddDataset(path)
			if err != nil {
				return false, err
			}
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
		if record.Path != path {
			record.Path = path
			UpdateRecord(record)
		}
		return true, nil
	}
	return false, nil
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
	datasetfilePath := filepath.Join(path, ".datasetinfo")
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
