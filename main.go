package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	inspectV := flag.Bool("inspect", false, "inspect existing records")
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
		log.Fatalf("can't init db, error: %v", err)
	}
	if *inspectV {
		err = inspect()
		if err != nil {
			log.Fatalf("fail to inspect, err: %v", err)
		}
		return
	}
	err = ScanFolders(appConfig.Root)
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
