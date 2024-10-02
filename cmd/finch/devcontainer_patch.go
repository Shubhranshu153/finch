// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package main denotes the entry point of finch CLI.
// TODO: Remove all instances of these calls once supported upstream
package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/runfinch/finch/pkg/command"
	"github.com/sirupsen/logrus"
)

// nerdctl outputs multiple id inspect call as multiple arrays of json which is not parsable by json.
//
//	Helper function helps to separate out into individual json array and a single json is reconstructed.
func extractOuterArrays(input string) []string {
	var arrays []string
	var bracketCount, start int

	for i, char := range input {
		switch char {
		case '[':
			if bracketCount == 0 {
				start = i
			}
			bracketCount++
		case ']':
			bracketCount--
			if bracketCount == 0 {
				arrays = append(arrays, input[start:i+1])
			}
		}
	}

	return arrays
}

func prettyPrintJSON(input string) {
	jsonArrays := extractOuterArrays(input)
	var mergedData []map[string]interface{}

	for i, jsonArray := range jsonArrays {
		var parsedArray []map[string]interface{}
		err := json.Unmarshal([]byte(jsonArray), &parsedArray)
		if err != nil {
			logrus.Error("Error parsing JSON at index: ", i, err)
			continue
		}

		if len(parsedArray) == 0 {
			continue
		}

		// its always is a single object
		firstObject := parsedArray[0]
		_, ok := firstObject["Config"].(map[string]interface{})
		if ok {
			if image, ok := firstObject["Image"].(string); ok {
				firstObject["Config"].(map[string]interface{})["Image"] = image
			}
		}

		if _, ok := firstObject["State"].(map[string]interface{}); ok {
			firstObject["State"].(map[string]interface{})["StartedAt"] = "0001-01-01T00:00:00Z"
		}

		mergedData = append(mergedData, firstObject)
	}

	finalJSON, err := json.MarshalIndent(mergedData, "", "  ")
	if err != nil {
		logrus.Error("Error marshaling final JSON: ", err)
		return
	}

	fmt.Println(string(finalJSON))
}

func inspectContainerOutputHandler(cmd command.Command) error {
	var stdoutBuf bytes.Buffer
	cmd.SetStdout(&stdoutBuf)
	getInspectType()

	err := cmd.Run()

	prettyPrintJSON(stdoutBuf.String())
	return err
}
