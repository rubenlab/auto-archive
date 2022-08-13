package main

import (
	"log"
	"os"
	"path/filepath"
)

func ScanFolders(rootPath string) error {
	scanLevel := appConfig.ScanLevel
	currentLevel := 1
	return scanFoldersInternal(rootPath, currentLevel, scanLevel)
}

func scanFoldersInternal(rootPath string, currentLevel int, scanLevel int) error {
	files, err := os.ReadDir(rootPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			// skip symlink folder
			continue
		}
		path := filepath.Join(rootPath, file.Name())
		isDataset, err := CreateIfDataset(path)
		if err != nil {
			return err
		}
		if isDataset {
			continue
		}
		if currentLevel >= scanLevel {
			err = AddDataset(path)
			if err != nil {
				log.Printf("error add dataset, error: %v", err)
			}
			continue
		}
		err = scanFoldersInternal(path, currentLevel+1, scanLevel)
		if err != nil {
			return err
		}
	}
	return nil
}
