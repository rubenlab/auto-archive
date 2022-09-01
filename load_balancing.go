package main

import (
	"database/sql"
	"time"
)

func LoadBalancing() error {
	scanInterval := appConfig.ScanInterval
	if scanInterval <= 1 {
		return nil
	}
	list, err := ListActiveRecords()
	if err != nil {
		return err
	}
	now := time.Now()
	for i, r := range list {
		addDays := (i % scanInterval) + 1
		r.ScanTime = sql.NullTime{
			Time:  now.AddDate(0, 0, -addDays),
			Valid: true,
		}
		err = UpdateRecord(&r)
		if err != nil {
			return err
		}
	}
	return nil
}
