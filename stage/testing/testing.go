package testing

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Netcracker/qubership-fluent-pipeline-tests/agent"
	"github.com/rodaine/table"
)

var (
	succeeded            = "\u001b[32mSucceeded\u001B[0m"
	failed               = "\u001b[31mFailed\u001B[0m"
	differencesStatus    = "\u001b[33;20mLog is processed, but with differences\u001B[0m"
	logNotFoundStatus    = "\u001b[31mLog is not found in output\u001B[0m"
	moreThanOneLogStatus = "\u001b[31mMore than one occupancies in output\u001B[0m"
)

func CompareLogs(ignoreFiles string, agent agent.Agent, modificationFuncs []RecordModifyFunc) {
	success, err := testJson(ignoreFiles, agent, modificationFuncs)
	if err != nil {
		slog.Error("Error occurred when reading output logs", "err", err)
		os.Exit(1)
	}
	if !success {
		slog.Error("Fluent pipeline test is failed. See logs")
		os.Exit(1)
	}
	slog.Info("Check finished successfully!")
}

func testJson(ignore string, agent agent.Agent, modificationFuncs []RecordModifyFunc) (bool, error) {
	success := true
	ignoreFiles := strings.Split(ignore, ",")

	outputLogFileName := agent.GetOutputFileName()
	if len(outputLogFileName) == 0 {
		slog.Error("Could not get filename for agent", "agent", agent)
		return false, nil
	}

	slog.Info("Reading actual log file...", "filename", outputLogFileName)

	actual, err := os.ReadFile(filepath.Join("output-logs", "actual", outputLogFileName)) //fluent-pipeline-test/output-logs
	if err != nil {
		return false, err
	}
	var actualJsonString []byte
	actualJsonString = append(actualJsonString, 91)
	actualJsonString = append(actualJsonString, actual...)
	actualJsonString = append(actualJsonString, 93)
	actualFinal := strings.ReplaceAll(string(actualJsonString), "}\n{", "},{")
	slog.Debug("Actual log file successfully read", "content", actualFinal)
	var resultJsonStruct []map[string]interface{}
	err = json.Unmarshal([]byte(actualFinal), &resultJsonStruct)
	if err != nil {
		return false, err
	}
	pathExpectedLogs := filepath.Join("output-logs", "expected") //fluent-pipeline-test/output-logs
	outputLogsDir := os.DirFS(pathExpectedLogs)
	slog.Info("Reading expected logs directory...", "dir", pathExpectedLogs)

	report := table.New("LOG ID", "STATUS", "DETAILS").WithHeaderSeparatorRow('-').WithPadding(5)
	notFoundLogIds := table.New("FILE PATH").WithPadding(5)
	printNotFoundLogIdsTable := false
	err = filepath.Walk(pathExpectedLogs, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".log.json") {
			_, expectedFile := filepath.Split(path)
			if contains(ignoreFiles, expectedFile) {
				slog.Info(fmt.Sprintf("Skipping file %s", expectedFile))
				return nil
			}
			expected, err := fs.ReadFile(outputLogsDir, expectedFile)
			if err != nil {
				success = false
				slog.Error("Error occurred while reading file", "path", path)
				return err
			}
			slog.Debug("Reading file", "path", path, "expectedFile", expectedFile)

			var expectedJsonStruct []map[string]interface{}
			err = json.Unmarshal(expected, &expectedJsonStruct)
			if err != nil {
				success = false
				return err
			}
			//compare actual and expected
			for _, record := range expectedJsonStruct {
				//parse logId and compare by logId
				if record["logId"] != nil {
					logId := record["logId"]
					var actualRecord map[string]interface{}
					var isDuplicated bool
					for _, actualRec := range resultJsonStruct {
						if actualRecord != nil && actualRec["logId"] == logId {
							isDuplicated = true
							break
						} else if actualRecord == nil && actualRec["logId"] == logId {
							actualRecord = actualRec
						}
					}
					if isDuplicated {
						success = false
						slog.Warn(fmt.Sprintf("Check logs from %q with logId=%s is failed. There were found more than one occupancies of the log in actual logs", expectedFile, logId))
						report.AddRow(logId, failed, moreThanOneLogStatus)
					} else if !isDuplicated && actualRecord != nil {
						if err := applyModificationFuncs(record, actualRecord, expectedFile, modificationFuncs); err != nil {
							slog.Error("could not apply modification function to records")
						}
						isEqual := reflect.DeepEqual(record, actualRecord)
						if isEqual {
							//everything is ok!
							report.AddRow(logId, succeeded)
							slog.Debug(fmt.Sprintf("Check logs from %q with logId=%s is successful: record parsed", expectedFile, logId))
						} else {
							//check failed
							success = false
							report.AddRow(logId, failed, differencesStatus)
							slog.Warn(fmt.Sprintf("Check logs from %q with logId=%s is failed. Expected log printed below", expectedFile, logId))
							err = printJsonRecord(fmt.Sprintf("%v", logId), record, true)
							if err != nil {
								slog.Error("Error occurred while printing log record", "err", err)
								return err
							}
							slog.Warn("Actual log printed below")
							err = printJsonRecord(fmt.Sprintf("%v", logId), actualRecord, false)
							if err != nil {
								slog.Error("Error occurred while printing log record", "err", err)
								return err
							}
						}
						actualRecord = nil
						continue
					} else {
						//check failed
						success = false
						report.AddRow(logId, failed, logNotFoundStatus)
						slog.Error(fmt.Sprintf("could not find logId with value %s in file %q with actual logs", logId, expectedFile))
					}
				} else {
					//check failed
					success = false
					printNotFoundLogIdsTable = true
					notFoundLogIds.AddRow("expectedFile")
					slog.Error(fmt.Sprintf("could not find %q in file %q with expected logs", "logId", expectedFile))
				}
			}
		}
		return nil
	})
	if err != nil {
		success = false
		return success, err
	}
	slog.Info("Check finished!")
	fmt.Printf("--- Report of %s pipeline testing ---", agent)
	fmt.Println()
	report.Print()

	if printNotFoundLogIdsTable {
		fmt.Printf("--- Files where %q was not found ---", "logId")
		fmt.Println()
		notFoundLogIds.Print()
	}

	return success, err
}
