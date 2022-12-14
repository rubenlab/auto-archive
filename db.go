package main

import (
	"bytes"
	"database/sql"
	"encoding/gob"

	bolt "go.etcd.io/bbolt"
)

const Bucket_Active = "active"
const Bucket_Archived = "archived"

type DatasetRecord struct {
	ID              string
	Path            string
	LastModifyTime  sql.NullTime // the latest modify time of any file in the folder
	ScanTime        sql.NullTime // when it's last scanned
	NoticedLeftDays int          // Reminder for archiving in NoticedLeftDays have been sent
	ArchiveTime     sql.NullTime // When record is archived
}

var currentDb *bolt.DB = nil

func initDb() (*bolt.DB, error) {
	db, err := bolt.Open(appConfig.DB, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(Bucket_Active))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(Bucket_Archived))
		return err
	})
	if err != nil {
		return nil, err
	}
	currentDb = db
	return db, nil
}

func AddRecord(record *DatasetRecord) error {
	err := currentDb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(Bucket_Active))
		data, err := encodeRecord(record)
		if err != nil {
			return err
		}
		bucket.Put([]byte(record.ID), data)
		return nil
	})
	return err
}

func UpdateRecord(record *DatasetRecord) error {
	return AddRecord(record)
}

func GetRecord(id string) (*DatasetRecord, error) {
	var record *DatasetRecord
	err := currentDb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(Bucket_Active))
		data := bucket.Get([]byte(id))
		if data == nil {
			return nil
		}
		recordEntity, err := decodeRecord(data)
		if err != nil {
			return err
		}
		record = recordEntity
		return nil
	})
	return record, err
}

func encodeRecord(record *DatasetRecord) ([]byte, error) {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(record)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeRecord(data []byte) (*DatasetRecord, error) {
	d := DatasetRecord{}
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func DeleteRecord(id string) error {
	err := currentDb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(Bucket_Active))
		bucket.Delete([]byte(id))
		return nil
	})
	return err
}

func SaveArchiveRecord(record *DatasetRecord) error {
	err := currentDb.Update(func(tx *bolt.Tx) error {
		activeBucket := tx.Bucket([]byte(Bucket_Active))
		activeBucket.Delete([]byte(record.ID))
		archiveBucket := tx.Bucket([]byte(Bucket_Archived))
		data, err := encodeRecord(record)
		if err != nil {
			return err
		}
		archiveBucket.Put([]byte(record.ID), data)
		return nil
	})
	return err
}

func ListActiveRecords() ([]DatasetRecord, error) {
	return listBucketRecords(Bucket_Active)
}

func ListArchivedRecords() ([]DatasetRecord, error) {
	return listBucketRecords(Bucket_Archived)
}

func listBucketRecords(bucketName string) ([]DatasetRecord, error) {
	list := make([]DatasetRecord, 0, 10)
	err := currentDb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		err := bucket.ForEach(func(k, v []byte) error {
			record, err := decodeRecord(v)
			if err != nil {
				return err
			}
			list = append(list, *record)
			return nil
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}
