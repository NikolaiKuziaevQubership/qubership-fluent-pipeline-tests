package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type RecordModifyFunc func(actual, expected map[string]interface{}, file string) error

func ignoreFluentdTimeFunc(ignoreFluentdTimeFiles string) RecordModifyFunc {
	ignoreFiles := strings.Split(ignoreFluentdTimeFiles, ",")
	return func(expected, actual map[string]interface{}, file string) error {
		if contains(ignoreFiles, file) {
			expected["fluentd_time"] = actual["fluentd_time"]
		}
		return nil
	}
}

func GetModificationFuncs(agent string, ignoreFluentdTimeFiles string) (rmFuncs []RecordModifyFunc) {
	if strings.EqualFold(agent, "fluentd") && len(ignoreFluentdTimeFiles) > 0 {
		rmFuncs = append(rmFuncs, ignoreFluentdTimeFunc(ignoreFluentdTimeFiles))
	}
	return
}

func applyModificationFuncs(record map[string]interface{}, actualRecord map[string]interface{}, file string, modificationFuncs []RecordModifyFunc) error {
	for _, applyFunc := range modificationFuncs {
		if err := applyFunc(record, actualRecord, file); err != nil {
			return err
		}
	}
	return nil
}

func contains(slc []string, el string) bool {
	for i := range slc {
		if el == slc[i] {
			return true
		}
	}
	return false
}

func printJsonRecord(logId string, record map[string]interface{}, expected bool) error {
	src, err := json.Marshal(record)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, src, "", "\t")
	if err != nil {
		return err
	}
	if expected {
		fmt.Printf("\u001B[32m--- Expected log. LogId=%s ---\u001B[0m", logId)
	} else {
		fmt.Printf("\u001B[33;20m--- Actual log. LogId=%s ---\u001B[0m", logId)
	}
	fmt.Println()
	fmt.Println(buf.String())
	return nil
}
