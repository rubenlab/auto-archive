package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nightlyone/lockfile"
)

func main() {
	inspectV := flag.Bool("inspect", false, "inspect existing records")
	loadBalance := flag.Bool("load-balance", false, "load balance existing records")
	flag.Parse()
	configFile := flag.Arg(0)
	if configFile == "" {
		fmt.Println("Please provide a config file, usage: autoarchive config.yml")
		os.Exit(1)
	}
	err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("can't load config, err: %v", err)
	}
	_, err = initDb()
	if err != nil {
		log.Fatalf("can't init db, error: %v\n", err)
	}
	// initialization finish

	if *inspectV {
		err = inspect()
		if err != nil {
			log.Fatalf("fail to inspect, err: %v", err)
		}
		return
	} else if *loadBalance {
		log.Println("start load balance")
		err = LoadBalancing()
		if err != nil {
			log.Fatalf("fail to load balance, err: %v", err)
		}
		log.Println("finish load balance")
		return
	}

	// init log, log after here will be written to config.LogFolder
	logCloser, logErr := initLog()
	if logErr != nil {
		log.Printf("init log error, error is: %v", logErr)
	}
	defer func() {
		if logCloser != nil {
			logCloser.Close()
		}
	}()

	// get pid lock, avoid concurrent execution
	pidLock, lockErr := tryLock()
	defer func() {
		if pidLock != nil {
			pidLock.Unlock()
		}
	}()
	if lockErr != nil {
		log.Println("failed to get lock, other autoarchive process is running")
		os.Exit(1)
	}

	// do auto archiving
	log.Println("start auto archive")
	autoArchive()
	log.Println("finish auto archive")
}

func autoArchive() {
	err := ScanFolders(appConfig.Root)
	if err != nil {
		log.Println(err)
	}
	scanResult, err := ScanRecords()
	if err != nil {
		log.Fatalf("error in scan records, error: %v", err)
	}
	err = SendNotice(scanResult)
	if err != nil {
		log.Printf("error send notice, error: %v", err)
	}
}

func tryLock() (*lockfile.Lockfile, error) {
	if appConfig.PidFile == "" {
		return nil, nil
	}
	fileLock, err := lockfile.New(appConfig.PidFile)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	err = fileLock.TryLock()

	return &fileLock, err
}
