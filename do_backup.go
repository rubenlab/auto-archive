package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func doBackup(path string) error {
	backupCommand := appConfig.BackupCommand
	if backupCommand == "" { // if there's no backup command, skip backup
		return nil
	}
	info, err := ReadDatasetinfo(path)
	if err != nil {
		return err
	}
	if info == nil {
		return errors.New(fmt.Sprintf("dataset file %s doesn't exists", DatasetFileName))
	}
	backupTime := info.BackupTime
	relativePaths, maxUpdateTime, fullUpdate, err := getBackupList(path, ".", backupTime)
	// maxUpdateTime must not be earlier than backupTime of the last scan
	if err != nil {
		return err
	}
	if backupTime.Valid && maxUpdateTime.Before(backupTime.Time) {
		maxUpdateTime = backupTime.Time
	}
	if len(relativePaths) == 0 {
		return nil
	}
	err = makeBackup(path, info, relativePaths, fullUpdate)
	if err != nil {
		return err
	}
	info.BackupTime = sql.NullTime{
		Time:  maxUpdateTime,
		Valid: true,
	}
	err = SaveDatasetInfo(path, info)
	return nil
}

func getBackupList(basePath string, relativePath string, backupTime sql.NullTime) ([]string, time.Time, bool, error) {
	absPath := filepath.Join(basePath, relativePath)
	dirs, err := os.ReadDir(absPath)
	if err != nil {
		return nil, time.Time{}, false, err
	}
	updatedPaths := make([]string, 0, len(dirs))
	maxUpdateTime := time.Time{}
	fullUpdate := true
	for _, dir := range dirs {
		info, e := dir.Info()
		if e != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			// skip symlink
			continue
		}
		if info.Name() == DatasetFileName { // skip datasetinfo file
			continue
		}

		if info.IsDir() {
			relPath := filepath.Join(relativePath, info.Name())
			subPaths, subMaxUpdateTime, subFullUpdate, err := getBackupList(basePath, relPath, backupTime)
			if err != nil {
				return nil, time.Time{}, false, err
			}
			if subFullUpdate { // the whole folder is updated
				if len(subPaths) > 0 { // if there's some update
					updatedPaths = append(updatedPaths, relPath)
				}
			} else {
				updatedPaths = append(updatedPaths, subPaths...)
				fullUpdate = false
			}
			if subMaxUpdateTime.After(maxUpdateTime) {
				maxUpdateTime = subMaxUpdateTime
			}
		} else {
			if backupTime.Valid {
				if !backupTime.Time.Before(info.ModTime()) { // file not modified
					fullUpdate = false
					continue
				}
			}
			// file updated
			relPath := filepath.Join(relativePath, info.Name())
			updatedPaths = append(updatedPaths, relPath)
		}
		if info.ModTime().After(maxUpdateTime) {
			maxUpdateTime = info.ModTime()
		}
	}
	// if it's a full update, change the subPaths to the folder itself.
	// But if it's the top folder, don't do this
	if len(updatedPaths) > 0 && fullUpdate && relativePath != "." {
		updatedPaths = make([]string, 0, 1)
		updatedPaths = append(updatedPaths, relativePath)
	}
	return updatedPaths, maxUpdateTime, fullUpdate, nil
}

func makeBackup(path string, info *Datasetinfo, relativePaths []string, fullUpdate bool) error {
	if len(relativePaths) == 0 {
		return nil
	}
	file, err := os.CreateTemp("", "")
	datawriter := bufio.NewWriter(file)
	for _, data := range relativePaths {
		_, err = datawriter.WriteString(data + "\n")
		if err != nil {
			return err
		}
	}

	datawriter.Flush()
	file.Close()
	if err != nil {
		return err
	}

	date := time.Now().Format("2006-01-02")
	backupCommand := appConfig.BackupCommand
	err = execBackupCommand(info.ID, path, file.Name(), date, backupCommand)
	os.Remove(file.Name())
	return err
}

func execBackupCommand(id string, dir string, file string, date string, backupCommand string) error {
	if backupCommand == "" {
		return nil
	}
	backupCommand = strings.Replace(backupCommand, "${id}", id, -1)
	backupCommand = strings.Replace(backupCommand, "${dir}", dir, -1)
	backupCommand = strings.Replace(backupCommand, "${file}", file, -1)
	backupCommand = strings.Replace(backupCommand, "${date}", date, -1)
	fields, err := getFields(backupCommand)
	if err != nil {
		return err
	}
	name := fields[0]
	args := fields[1:]
	cmd := exec.Command(name, args...)
	rc, logErr := getBackupWriter(dir, id)
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

func getBackupWriter(path string, id string) (io.WriteCloser, error) {
	logFileName := "backup_" + id + ".log"
	title := fmt.Sprintf("folder path: %s\n", path)
	return getLogWriter(logFileName, title)
}
