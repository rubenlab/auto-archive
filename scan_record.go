package main

import (
	"database/sql"
	"log"
	"os"
	"sort"
	"time"

	"github.com/gammazero/workerpool"
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

type ScanResultModifier struct {
	Error          *ScanError
	Notice         *ArchiveNotice
	ArchivedFolder *ArchivedFolder
}

func (m *ScanResultModifier) modify(result *ScanResult) {
	if m.Error != nil {
		result.Errors = append(result.Errors, *m.Error)
	} else if m.Notice != nil {
		result.Notices = append(result.Notices, *m.Notice)
	} else if m.ArchivedFolder != nil {
		result.ArchivedFolders = append(result.ArchivedFolders, *m.ArchivedFolder)
	}
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
	c := make(chan ScanResultModifier)
	finishChan := make(chan int)
	go func(result *ScanResult, c *chan ScanResultModifier, finishChan *chan int) {
		for v := range *c {
			v.modify(result)
		}
		close(*finishChan)
	}(&scanResult, &c, &finishChan)
	wp := workerpool.New(appConfig.cores)
	for _, record := range records {
		wp.Submit(func() {
			scanRecord(&record, &c)
		})
	}
	wp.StopWait()
	close(c)
	// wait for the finish signal
	<-finishChan
	return &scanResult, nil
}

func scanRecord(record *DatasetRecord, c *chan ScanResultModifier) {
	id := record.ID
	path := record.Path
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			DeleteRecord(record.ID)
		} else {
			log.Printf("failed to open directory, error: %v", err)
			addErrResult(id, path, err, c)
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
		log.Printf("failed to scan update time, error: %v", err)
		addErrResult(id, path, err, c)
	}
	record.LastModifyTime = sql.NullTime{
		Time:  lastUpdateTime,
		Valid: true,
	}
	record.ScanTime = sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	afterScan(record, c)
}

func addErrResult(id string, path string, err error, c *chan ScanResultModifier) {
	scanErr := ScanError{
		ID:   id,
		Path: path,
		Msg:  err.Error(),
	}
	*c <- ScanResultModifier{Error: &scanErr}
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
func afterScan(record *DatasetRecord, c *chan ScanResultModifier) {
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
		err := doArchive(path, id)
		if err != nil {
			log.Printf("failed to archive, error: %v", err)
			addErrResult(id, path, err, c)
			UpdateRecord(record)
			return
		}
		err = SaveArchiveRecord(record)
		if err != nil {
			log.Printf("failed to save archive record, error: %v", err)
			addErrResult(id, path, errors.Wrap(err, "failed to save archived record"), c)
			return
		}
		archivedFolder := ArchivedFolder{
			ID:   id,
			Path: path,
		}
		*c <- ScanResultModifier{ArchivedFolder: &archivedFolder}
		return
	} else { // make incremental backups
		err := doBackup(path)
		if err != nil {
			log.Printf("failed to do backup, error: %v", err)
			addErrResult(id, path, err, c)
			UpdateRecord(record)
			return
		}
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
				*c <- ScanResultModifier{Notice: &notice}
				record.NoticedLeftDays = noticeLeftDays
				UpdateRecord(record)
				return
			}
		}
	}

	// For folders don't need to do anything, just up date the record.
	UpdateRecord(record)
}
