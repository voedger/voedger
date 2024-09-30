/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author: Alisher Nurmanov
 */

package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/voedger/voedger/pkg/goutils/logger"

	"github.com/voedger/voedger/pkg/ctrlloop"
)

type InputControlMessage struct {
	Type  string
	Value json.RawMessage
}

var inputStreamReadingInterval time.Duration = 0

func newRunEdgerCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "run",
		Short: "Runs edger and processes SP values from stdin",
		Run: func(c *cobra.Command, args []string) {
			commandInCh := make(chan ctrlloop.ControlMessage[string, CommandSP])
			commandCtrlloopWaitFunc := ctrlloop.New(CommandController, CommandReporter, NumCommandControllerRoutines, commandInCh, time.Now)
			defer commandCtrlloopWaitFunc()
			defer close(commandInCh)

			dockerInCh := make(chan ctrlloop.ControlMessage[string, DockerSP])
			dockerCtrlloopWaitFunc := ctrlloop.New(DockerController, DockerReporter, NumDockerControllerRoutines, dockerInCh, time.Now)
			defer dockerCtrlloopWaitFunc()
			defer close(dockerInCh)

			runEdger(c.Context(), os.Stdin, commandInCh, dockerInCh)
		},
	}

	return &cmd
}

func runEdger(ctx context.Context, r io.Reader, commandInCh chan ctrlloop.ControlMessage[string, CommandSP], dockerInCh chan ctrlloop.ControlMessage[string, DockerSP]) {
	decoder := json.NewDecoder(r)

	var decodingErr error
	for !errors.Is(decodingErr, io.EOF) && ctx.Err() == nil {
		var input InputControlMessage
		decodingErr = decoder.Decode(&input)
		if decodingErr != nil {
			if !errors.Is(decodingErr, io.EOF) {
				logger.Verbose("error decoding JSON:", decodingErr)
			}
			continue
		}

		switch input.Type {
		case SPTypeCommand:
			m := getControlMessage[string, CommandSP](input)
			if m != nil {
				commandInCh <- *m
			}
		case SPTypeDocker:
			m := getControlMessage[string, DockerSP](input)
			if m != nil {
				dockerInCh <- *m
			}
		default:
			logger.Verbose("unknown sp type: " + input.Type)
		}
		time.Sleep(inputStreamReadingInterval)
	}
}

func decodeControlMessage[Key comparable, SP any](input InputControlMessage) *ctrlloop.ControlMessage[Key, SP] {
	var sp ctrlloop.ControlMessage[Key, SP]
	if err := json.Unmarshal(input.Value, &sp); err != nil {
		logger.Verbose("error decoding")
		return nil
	}
	return &sp
}

func getControlMessage[Key comparable, SP any](input InputControlMessage) *ctrlloop.ControlMessage[Key, SP] {
	switch input.Type {
	case SPTypeCommand:
		return decodeControlMessage[Key, SP](input)
	case SPTypeDocker:
		return decodeControlMessage[Key, SP](input)
	default:
		logger.Verbose("unknown sp type: " + input.Type)
		return nil
	}
}
