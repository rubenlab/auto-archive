package main

import (
	"io/fs"
	"log"
	"path/filepath"
	"time"
)

func scanUpdateTime(path string) (time.Time, error) {
	var lastUpdateTime time.Time
	filepath.Walk(path, func(path string, f fs.FileInfo, err error) error {
		if f.Name() == DatasetFileName {
			return nil
		}
		modifyTime := f.ModTime()
		if lastUpdateTime.IsZero() || modifyTime.After(lastUpdateTime) {
			lastUpdateTime = modifyTime
		}
		if err != nil {
			log.Printf("error when Wals through folder %s, the error is: %v", path, err)
		}
		return nil
	})
	return lastUpdateTime, nil
}
