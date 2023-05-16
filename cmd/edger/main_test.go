/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Alisher Nurmanov
 */

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mockStdin(t *testing.T, dummyInput string) (restoreStdinFunc func(), err error) {
	t.Helper()

	oldOsStdin := os.Stdin

	tmpFile, err := os.CreateTemp(t.TempDir(), t.Name())
	if err != nil {
		return nil, err
	}

	content := []byte(dummyInput)

	if _, err := tmpFile.Write(content); err != nil {
		return nil, err
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, err
	}

	// Set stdin to the temp file
	os.Stdin = tmpFile

	return func() {
		// clean up
		os.Stdin = oldOsStdin
		_ = os.Remove(tmpFile.Name())
	}, nil
}

func TestExecRootCmd(t *testing.T) {
	// Test case: Provide valid arguments and version
	args := []string{"edger", "run"}
	ver := "1.0.0"

	projectName := "my"
	testData := InputControlMessage{
		Type: "docker",
		Value: json.RawMessage(fmt.Sprintf(`{
			"Key": "%s",
			"SP": {
				"Version": "1.0",
				"ComposeText": "version: \"3.7\"\nservices:\n  redis:\n    image: 'redis:7.0.11-alpine'\n    restart: always\n  nginx:\n    image: 'nginx:1.23.4'\n    restart: always\n"
			}
		}`, projectName)),
	}
	jsonData, err := json.Marshal(testData)
	require.NoError(t, err)

	restoreStdinFunc, err := mockStdin(t, string(jsonData))
	if err != nil {
		t.Fatal(err)
	}

	defer restoreStdinFunc()

	expectedNewState := dockerContainerInfoList{
		{
			Name:  fmt.Sprintf("%s-redis-1", projectName),
			Image: "redis:7.0.11-alpine",
			IsUp:  true,
		},
		{
			Name:  fmt.Sprintf("%s-nginx-1", projectName),
			Image: "nginx:1.23.4",
			IsUp:  true,
		},
	}

	err = cleanUp(projectName)
	require.NoError(t, err)

	inputStreamReadingInterval = 10 * time.Millisecond
	err = execRootCmd(args, ver)
	require.NoError(t, err, "Expected no error")

	newState, err := dockerContainers(projectName)
	require.NoError(t, err)

	sort.Sort(expectedNewState)
	sort.Sort(newState)
	require.Equal(t, len(expectedNewState), len(newState))
	require.Equal(t, expectedNewState, newState)

	err = cleanUp(projectName)
	require.NoError(t, err)
}
