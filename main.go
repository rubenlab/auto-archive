package main

import (
	"flag"
	"log"
)

func main() {
	flag.Parse()
	configFile := flag.Arg(0)
	loadConfig(configFile)
	err := ScanFolders(appConfig.Root)
	if err != nil {
		log.Println(err)
	}
	scanResult, err := ScanRecords()
	if err != nil {
		log.Fatal(err)
	}
	err = SendNotice(scanResult)
	if err != nil {
		log.Printf("error send notice, error: %v", err)
	}
}
