/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ctrlloop

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/untillpro/goutils/logger"
)

func Test_BasicUsage(t *testing.T) {
	// TODO: add serial number related tests
	logger.SetLogLevel(logger.LogLevelVerbose)

	mockGetNextTimeFunc := func(cronSchedule string, startTimeTolerance time.Duration, nowTime time.Time) time.Time {
		return nowTime
	}

	nextStartTimeFunc = mockGetNextTimeFunc

	tests := []struct {
		name                string
		numReportedMessages int
		controller          ControllerFunction[string, int, string, int]
		messages            []OriginalMessage[string, int]
	}{
		{
			name:                "3 messages:A->B->C",
			numReportedMessages: 3,
			controller: func(key string, sp int, state string) (newState *string, pv *int, startTime *time.Time) {
				logger.Verbose("controllerFunc")
				v := 1
				pv = &v
				return nil, pv, nil
			},
			messages: []OriginalMessage[string, int]{
				{
					Key:                `A`,
					SP:                 0,
					CronSchedule:       `*/1 * * * *`,
					StartTimeTolerance: 5 * time.Second,
				},
				{
					Key:                `B`,
					SP:                 1,
					CronSchedule:       `now`,
					StartTimeTolerance: 5 * time.Second,
				},
				{
					Key:                `C`,
					SP:                 2,
					CronSchedule:       `*/1 * * * *`,
					StartTimeTolerance: 5 * time.Second,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wg := sync.WaitGroup{}

			mtx := sync.Mutex{}
			reportDB := make([]struct {
				Key string
				PV  *int
			}, 0)

			reporterFunc := func(key string, pv *int) (err error) {
				mtx.Lock()
				defer mtx.Unlock()

				logger.Verbose("reporterFunc")

				defer wg.Done()
				reportDB = append(reportDB, struct {
					Key string
					PV  *int
				}{Key: key, PV: pv})
				return nil
			}

			ch := make(chan OriginalMessage[string, int])

			New(test.controller, reporterFunc, 5, ch, time.Now)

			wg.Add(test.numReportedMessages)

			for _, m := range test.messages {
				ch <- OriginalMessage[string, int]{
					Key:                m.Key,
					SP:                 m.SP,
					CronSchedule:       m.CronSchedule,
					StartTimeTolerance: m.StartTimeTolerance,
				}
			}

			wg.Wait()

			close(ch)

			assert.GreaterOrEqual(t, test.numReportedMessages, len(reportDB))
		})
	}
}

func Test_Scheduler(t *testing.T) {
	mockGetNextTimeFunc := func(cronSchedule string, startTimeTolerance time.Duration, nowTime time.Time) time.Time {
		return nowTime
	}

	nextStartTimeFunc = mockGetNextTimeFunc

	tests := []struct {
		name                 string
		originalMessages     []OriginalMessage[string, int]
		messagesFromRepeater []scheduledMessage[string, int, struct{}]
	}{
		{
			name: `in<-A,B,C;repeat<-D,E`,
			originalMessages: []OriginalMessage[string, int]{
				{
					Key:                `A`,
					SP:                 0,
					CronSchedule:       `*/1 * * * *`,
					StartTimeTolerance: 5 * time.Second,
				},
				{
					Key:                `B`,
					SP:                 1,
					CronSchedule:       `now`,
					StartTimeTolerance: 5 * time.Second,
				},
				{
					Key:                `C`,
					SP:                 2,
					CronSchedule:       `*/1 * * * *`,
					StartTimeTolerance: 5 * time.Second,
				},
			},
			messagesFromRepeater: []scheduledMessage[string, int, struct{}]{
				{
					Key:          `D`,
					SP:           1,
					serialNumber: 4,
					StartTime:    time.Now(),
				},
				{
					Key:          `E`,
					SP:           1,
					serialNumber: 5,
					StartTime:    time.Now(),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inCh := make(chan OriginalMessage[string, int])
			dedupInCh := make(chan statefulMessage[string, int, struct{}])
			repeatCh := make(chan scheduledMessage[string, int, struct{}], 3)

			go scheduler(inCh, dedupInCh, repeatCh, time.Now)

			go testMessagesWriter(inCh, test.originalMessages)

			go testMessagesWriter(repeatCh, test.messagesFromRepeater)

			maxSerialNumber := uint64(0)
			messageCounter := 0
			for m := range dedupInCh {
				if m.serialNumber > maxSerialNumber {
					maxSerialNumber = m.serialNumber
				}
				messageCounter++
				if messageCounter == len(test.originalMessages)+len(test.messagesFromRepeater) {
					break
				}
			}

			// closing channels
			close(inCh)
			close(repeatCh)

			require.Equal(t, uint64(len(test.originalMessages)+len(test.messagesFromRepeater)), maxSerialNumber)
		})
	}
}

func Test_Dedupin(t *testing.T) {
	mockGetNextTimeFunc := func(cronSchedule string, startTimeTolerance time.Duration, nowTime time.Time) time.Time {
		return nowTime
	}

	nextStartTimeFunc = mockGetNextTimeFunc

	tests := []struct {
		name          string
		inProcessKeys []string
		messages      []statefulMessage[string, int, struct{}]
	}{
		{
			name:          `dedupIn<-A,B,B,C`,
			inProcessKeys: []string{``},
			messages: []statefulMessage[string, int, struct{}]{
				{
					Key:          `A`,
					SP:           0,
					serialNumber: 1,
				},
				{
					Key:          `B`,
					SP:           1,
					serialNumber: 1,
				},
				{
					Key:          `B`,
					SP:           2,
					serialNumber: 1,
				},
				{
					Key:          `C`,
					SP:           2,
					serialNumber: 1,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			InProcess := sync.Map{}
			dedupInCh := make(chan statefulMessage[string, int, struct{}])
			callerCh := make(chan statefulMessage[string, int, struct{}])
			repeatCh := make(chan scheduledMessage[string, int, struct{}], 3)

			go dedupIn(dedupInCh, callerCh, repeatCh, &InProcess, time.Now)

			go testMessagesWriter(dedupInCh, test.messages)

			messageCounter := 0
			for messageCounter < len(test.messages) {
				select {
				case <-callerCh:
					messageCounter++
				case <-repeatCh:
					messageCounter++
				}
			}

			// closing channels
			close(dedupInCh)
			close(repeatCh)
		})
	}
}

func Test_Repeater(t *testing.T) {
	mockGetNextTimeFunc := func(cronSchedule string, startTimeTolerance time.Duration, nowTime time.Time) time.Time {
		return nowTime
	}

	nextStartTimeFunc = mockGetNextTimeFunc

	now := time.Now()
	pv := 1

	tests := []struct {
		name          string
		inProcessKeys []string
		messages      []answer[string, int, int, struct{}]
	}{
		{
			name:          `repeater<-A,B,B,C`,
			inProcessKeys: []string{``},
			messages: []answer[string, int, int, struct{}]{
				{
					Key:          `A`,
					SP:           0,
					serialNumber: 1,
					StartTime:    &now,
					PV:           nil,
				},
				{
					Key:          `B`,
					SP:           1,
					serialNumber: 1,
					StartTime:    &now,
					PV:           nil,
				},
				{
					Key:          `B`,
					SP:           2,
					serialNumber: 1,
					StartTime:    nil,
					PV:           &pv,
				},
				{
					Key:          `C`,
					SP:           2,
					serialNumber: 1,
					StartTime:    nil,
					PV:           &pv,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repeaterCh := make(chan answer[string, int, int, struct{}])
			repeatCh := make(chan scheduledMessage[string, int, struct{}], 3)
			reporterCh := make(chan reportInfo[string, int])

			go repeater(repeaterCh, repeatCh, reporterCh)

			go testMessagesWriter(repeaterCh, test.messages)

			messageCounter := 0
			for messageCounter < len(test.messages) {
				select {
				case <-reporterCh:
					messageCounter++
				case <-repeatCh:
					messageCounter++
				}
			}

			// closing channels
			close(repeaterCh)
		})
	}
}

func testMessagesWriter[T any](ch chan<- T, arr []T) {
	for _, m := range arr {
		ch <- m
	}
}
