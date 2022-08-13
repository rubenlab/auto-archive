package main

import (
	"database/sql"
	"os"
	"sort"
	"time"

	"github.com/pkg/errors"
)

type ScanResult struct {
	Errors          []ScanError
	Notices         []ArchiveNotice
	ArchivedFolders []ArchivedFolder
}

type ScanError struct {
	ID   string
	Path string
	Msg  string
}

func (e *ScanError) Error() string {
	return e.Msg
}

type ArchiveNotice struct {
	ID                string
	Path              string
	DaysBeforeArchive int
}

type ArchivedFolder struct {
	ID   string
	Path string
}

func ScanRecords() (*ScanResult, error) {
	records, err := ListActiveRecords()
	if err != nil {
		return nil, err
	}
	scanResult := ScanResult{
		Errors:          make([]ScanError, 0, 10),
		Notices:         make([]ArchiveNotice, 0, 10),
		ArchivedFolders: make([]ArchivedFolder, 0, 10),
	}
	for _, record := range records {
		scanRecord(&record, &scanResult)
	}
	return &scanResult, nil
}

func scanRecord(record *DatasetRecord, result *ScanResult) {
	id := record.ID
	path := record.Path
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			DeleteRecord(record.ID)
		} else {
			addErrResult(id, path, err, result)
		}
	}
	if !fi.IsDir() {
		DeleteRecord(record.ID)
	}
	if !isShouldScan(record) {
		return
	}
	lastUpdateTime, err := scanUpdateTime(path)
	if err != nil {
		addErrResult(id, path, err, result)
	}
	record.LastModifyTime = sql.NullTime{
		Time:  lastUpdateTime,
		Valid: true,
	}
	record.ScanTime = sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	afterScan(record, result)
}

func addErrResult(id string, path string, err error, result *ScanResult) {
	scanErr := ScanError{
		ID:   id,
		Path: path,
		Msg:  err.Error(),
	}
	result.Errors = append(result.Errors, scanErr)
}

// if a directory is never scanned, or
// if a directory is not scanned for ScanInterval days, or
// if a directory should be archived today, or
// if a notice should be sent today, this folder should be scan, and return true
func isShouldScan(record *DatasetRecord) bool {
	scanTime := record.ScanTime
	if !scanTime.Valid {
		return true
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	scanDate := time.Date(scanTime.Time.Year(), scanTime.Time.Month(), scanTime.Time.Day(), 0, 0, 0, 0, scanTime.Time.Location())
	unscanDays := int(today.Sub(scanDate).Hours() / 24)
	if unscanDays >= appConfig.ScanInterval {
		return true
	}
	lastModifyTime := record.LastModifyTime
	if lastModifyTime.Valid {
		archiveInterval := appConfig.ArchiveInterval
		lastModifyDate := time.Date(lastModifyTime.Time.Year(), lastModifyTime.Time.Month(), lastModifyTime.Time.Day(), 0, 0, 0, 0, lastModifyTime.Time.Location())
		unchangeDays := int(today.Sub(lastModifyDate).Hours() / 24)
		leftDays := archiveInterval - unchangeDays
		if leftDays <= 0 { // if folder need to be archived today, rescan to check if there're new changes
			return true
		}
		noticeBefore := appConfig.NoticeBefore
		if noticeBefore != nil && len(noticeBefore) > 0 {
			noticedLeftDays := record.NoticedLeftDays
			sort.Sort(sort.Reverse(sort.IntSlice(noticeBefore)))
			for _, noticeLeftDays := range noticeBefore {
				if noticedLeftDays > 0 && noticedLeftDays <= noticeLeftDays { // This notice is already sent, skip it.
					continue
				}
				if leftDays <= noticeLeftDays { // When a notice should be triggered, rescan to check if there're new changes
					return true
				}
			}
		}
	}
	return false
}

// after scan and update lastModify time,
// send notice or do archive
func afterScan(record *DatasetRecord, result *ScanResult) {
	id := record.ID
	path := record.Path
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastModifyTime := record.LastModifyTime.Time
	lastModifyDate := time.Date(lastModifyTime.Year(), lastModifyTime.Month(), lastModifyTime.Day(), 0, 0, 0, 0, lastModifyTime.Location())
	unchangeDays := int(today.Sub(lastModifyDate).Hours() / 24)
	archiveInterval := appConfig.ArchiveInterval
	leftDays := archiveInterval - unchangeDays

	// Archive the directory and move the record
	if leftDays <= 0 {
		err := doArchive(path)
		if err != nil {
			addErrResult(id, path, err, result)
			UpdateRecord(record)
			return
		}
		err = SaveArchiveRecord(record)
		if err != nil {
			addErrResult(id, path, errors.Wrap(err, "failed to save archived record"), result)
		}
		archivedFolder := ArchivedFolder{
			ID:   id,
			Path: path,
		}
		result.ArchivedFolders = append(result.ArchivedFolders, archivedFolder)
		return
	}

	// check if notice should send, add it to the result object.
	// update the record to mark the notice is sent
	noticeBefore := appConfig.NoticeBefore
	if noticeBefore != nil && len(noticeBefore) > 0 {
		noticedLeftDays := record.NoticedLeftDays
		sort.Sort(sort.Reverse(sort.IntSlice(noticeBefore)))
		for _, noticeLeftDays := range noticeBefore {
			if noticedLeftDays > 0 && noticedLeftDays <= noticeLeftDays { // This notice is already sent, skip it.
				continue
			}
			if leftDays <= noticeLeftDays { // When a notice should be triggered, rescan to check if there're new changes
				notice := ArchiveNotice{
					ID:                id,
					Path:              path,
					DaysBeforeArchive: noticeLeftDays,
				}
				result.Notices = append(result.Notices, notice)
				record.NoticedLeftDays = noticeLeftDays
				UpdateRecord(record)
				return
			}
		}
	}

	// For folders don't need to do anything, just up date the record.
	UpdateRecord(record)
}
