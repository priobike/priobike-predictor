package history

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

// Lookup all .json files in the static path and write them into a json file.
// This serves as an index for the cycle analyzer.
func buildIndexFile() {
	// Lookup all .json files in the static path.
	info, err := ioutil.ReadDir(staticPath)
	if err != nil {
		panic(err)
	}
	jsonFiles := make([]string, 0)
	for _, file := range info {
		if file.IsDir() {
			continue
		}
		if file.Name()[len(file.Name())-5:] != ".json" {
			continue
		}
		if file.Name() == "index.json" {
			continue
		}
		jsonFiles = append(jsonFiles, file.Name())
	}
	// Write the json files into a json file.
	jsonBytes, err := json.Marshal(jsonFiles)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s/index.json", staticPath), jsonBytes, 0644)
	if err != nil {
		panic(err)
	}
}

// Build the index file periodically.
func UpdateIndex() {
	for {
		buildIndexFile()
		time.Sleep(10 * time.Second)
	}
}
