/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Alisher Nurmanov
 */

package main

import "time"

// Command related types
type (
	CommandSP struct {
		Cmd  string
		Args []string
	}
	CommandPV struct {
		Cmd      string
		Args     []string
		Stdout   string
		Stderr   string
		ExitCode int
	}
	CommandState struct{}
)

// Docker related types
type (
	DockerSP struct {
		ComposeText string
		Version     string
	}
	DockerPV struct {
		Err         error
		Version     string
		AttemptTime time.Time
	}
	DockerState struct{}

	dockerContainerFullInfo struct {
		Command      string `json:"Command"`
		CreatedAt    string `json:"CreatedAt"`
		ID           string `json:"ID"`
		Image        string `json:"Image"`
		Labels       string `json:"Labels"`
		LocalVolumes string `json:"LocalVolumes"`
		Mounts       string `json:"Mounts"`
		Names        string `json:"Names"`
		Networks     string `json:"Networks"`
		Ports        string `json:"Ports"`
		RunningFor   string `json:"RunningFor"`
		Size         string `json:"Size"`
		State        string `json:"State"`
		Status       string `json:"Status"`
	}
	dockerContainerInfoList []dockerContainerInfo

	dockerContainerInfo struct {
		Name  string
		Image string
		IsUp  bool
	}
)

func (a dockerContainerInfoList) Len() int           { return len(a) }
func (a dockerContainerInfoList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a dockerContainerInfoList) Less(i, j int) bool { return a[i].Name < a[j].Name }

func newDockerPV(err error, version string, attemptTime time.Time) *DockerPV {
	return &DockerPV{
		Err:         err,
		Version:     version,
		AttemptTime: attemptTime,
	}
}
