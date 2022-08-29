package main

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type InspectResult struct {
	Active   []DatasetRecord
	Archived []DatasetRecord
}

func inspect() error {
	active, err := ListActiveRecords()
	if err != nil {
		return errors.Wrap(err, "error reading active records")
	}
	archived, err := ListArchivedRecords()
	if err != nil {
		return errors.Wrap(err, "error reading archived records")
	}
	result := InspectResult{
		Active:   active,
		Archived: archived,
	}
	b, err := json.MarshalIndent(&result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
