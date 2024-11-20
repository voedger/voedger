/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ctrlloop

import (
	"container/list"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/goutils/logger"
)

func Test_BasicUsage(t *testing.T) {
	logger.SetLogLevel(logger.LogLevelVerbose)
	defer logger.SetLogLevel(logger.LogLevelInfo)

	mockGetNextTimeFunc := func(cronSchedule string, startTimeTolerance time.Duration, nowTime time.Time) time.Time {
		return nowTime
	}

	nextStartTimeFunc = mockGetNextTimeFunc

	tests := []struct {
		name                string
		numReportedMessages int
		controller          ControllerFunction[string, int, string, int]
		messages            []ControlMessage[string, int]
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
			messages: []ControlMessage[string, int]{
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

			inCh := make(chan ControlMessage[string, int])

			waitFunc := New(test.controller, reporterFunc, 5, inCh, time.Now)

			wg.Add(test.numReportedMessages)

			for _, m := range test.messages {
				inCh <- ControlMessage[string, int]{
					Key:                m.Key,
					SP:                 m.SP,
					CronSchedule:       m.CronSchedule,
					StartTimeTolerance: m.StartTimeTolerance,
				}
			}

			wg.Wait()

			close(inCh)

			waitFunc()

			assert.GreaterOrEqual(t, test.numReportedMessages, len(reportDB))
		})
	}
}

// nolint
func Test_SchedulerOnIn(t *testing.T) {
	alwaysNowFunc := func(cronSchedule string, startTimeTolerance time.Duration, nowTime time.Time) time.Time {
		return nowTime
	}

	var testNowTime = time.Date(2023, 4, 20, 00, 00, 00, 0, time.Now().Location())

	tests := []struct {
		name                    string
		originalMessages        []ControlMessage[string, int]
		scheduledItems          []scheduledMessage[string, int, struct{}]
		nextStartTimeFunc       nextStartTimeFunction
		nowTime                 time.Time
		expectedResultKeys      []string
		expectedMaxSerialNumber uint64
		expectedTopStartTime    time.Time
	}{
		{
			name: `2 keys`,
			originalMessages: []ControlMessage[string, int]{
				{
					Key: `A`,
					SP:  0,
				},
				{
					Key: `B`,
					SP:  4,
				},
			},
			scheduledItems:          []scheduledMessage[string, int, struct{}]{},
			nextStartTimeFunc:       alwaysNowFunc,
			nowTime:                 testNowTime,
			expectedResultKeys:      []string{`A`, `B`},
			expectedMaxSerialNumber: 1,
			expectedTopStartTime:    testNowTime,
		},
		{
			name: `2 keys, 1 key in ScheduledItems`,
			originalMessages: []ControlMessage[string, int]{
				{
					Key: `A`,
					SP:  0,
				},
				{
					Key: `B`,
					SP:  4,
				},
			},
			scheduledItems: []scheduledMessage[string, int, struct{}]{
				{
					Key:          `A`,
					SP:           0,
					serialNumber: 10,
					StartTime:    testNowTime,
				},
			},
			nextStartTimeFunc:       alwaysNowFunc,
			nowTime:                 testNowTime,
			expectedResultKeys:      []string{`A`, `B`},
			expectedMaxSerialNumber: 10,
			expectedTopStartTime:    testNowTime,
		},
		{
			name: `invalid CronSchedule`,
			originalMessages: []ControlMessage[string, int]{
				{
					Key:          `A`,
					SP:           0,
					CronSchedule: `QWERTY`,
				},
			},
			nowTime:                 testNowTime,
			scheduledItems:          []scheduledMessage[string, int, struct{}]{},
			nextStartTimeFunc:       getNextStartTime,
			expectedResultKeys:      []string{`A`},
			expectedMaxSerialNumber: 0,
			expectedTopStartTime:    testNowTime,
		},
		{
			name: `CronSchedule * 1 * * *, tolerance zero`,
			originalMessages: []ControlMessage[string, int]{
				{
					Key:          `A`,
					SP:           0,
					CronSchedule: `* 1 * * *`,
				},
			},
			scheduledItems:          []scheduledMessage[string, int, struct{}]{},
			nextStartTimeFunc:       getNextStartTime,
			nowTime:                 testNowTime,
			expectedResultKeys:      []string{`A`},
			expectedMaxSerialNumber: 0,
			expectedTopStartTime:    testNowTime.Add(1 * time.Hour),
		},
		{
			name: `CronSchedule 0 0 * * *, tolerance 5 min`,
			originalMessages: []ControlMessage[string, int]{
				{
					Key:                `A`,
					SP:                 0,
					CronSchedule:       `0 0 * * *`,
					StartTimeTolerance: 5 * time.Minute,
				},
			},
			scheduledItems:          []scheduledMessage[string, int, struct{}]{},
			nextStartTimeFunc:       getNextStartTime,
			nowTime:                 testNowTime.Add(299 * time.Second),
			expectedResultKeys:      []string{`A`},
			expectedMaxSerialNumber: 0,
			expectedTopStartTime:    testNowTime,
		},
		{
			name: `CronSchedule 0 0 * * *, tolerance 5 min, 1 second delay`,
			originalMessages: []ControlMessage[string, int]{
				{
					Key:                `A`,
					SP:                 0,
					CronSchedule:       `0 0 * * *`,
					StartTimeTolerance: 5 * time.Minute,
				},
			},
			scheduledItems:          []scheduledMessage[string, int, struct{}]{},
			nextStartTimeFunc:       getNextStartTime,
			nowTime:                 testNowTime.Add(301 * time.Second),
			expectedResultKeys:      []string{`A`},
			expectedMaxSerialNumber: 0,
			expectedTopStartTime:    testNowTime.Add(24 * time.Hour),
		},
		{
			name: `the second message scheduled before the first one`,
			originalMessages: []ControlMessage[string, int]{
				{
					Key:          `A`,
					SP:           0,
					CronSchedule: `0 11 * * *`,
				},
				{
					Key:          `B`,
					SP:           4,
					CronSchedule: `0 10 * * *`,
				},
			},
			scheduledItems:          []scheduledMessage[string, int, struct{}]{},
			nextStartTimeFunc:       getNextStartTime,
			nowTime:                 testNowTime,
			expectedResultKeys:      []string{`B`, `A`},
			expectedMaxSerialNumber: 1,
			expectedTopStartTime:    testNowTime.Add(10 * time.Hour),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nextStartTimeFunc = test.nextStartTimeFunc

			initList := list.New()
			for _, i := range test.scheduledItems {
				initList.PushBack(i)
			}

			schedulerObj := newScheduler[string, int, struct{}](initList)

			for i, m := range test.originalMessages {
				schedulerObj.OnIn(uint64(i), m, test.nowTime)
			}

			maxSerialNumber := uint64(0)
			resultKeys := make([]string, 0)
			for element := schedulerObj.scheduledItems.Front(); element != nil; element = element.Next() {
				m := element.Value.(scheduledMessage[string, int, struct{}])
				if m.serialNumber > maxSerialNumber {
					maxSerialNumber = m.serialNumber
				}
				resultKeys = append(resultKeys, m.Key)
			}

			top := schedulerObj.scheduledItems.Front().Value.(scheduledMessage[string, int, struct{}])
			require.Equal(t, test.expectedTopStartTime, top.StartTime)
			require.Equal(t, test.expectedResultKeys, resultKeys)
			require.Equal(t, test.expectedMaxSerialNumber, maxSerialNumber)
		})
	}
}

func Test_SchedulerOnRepeat(t *testing.T) {
	var testNowTime = time.Date(2023, 4, 20, 00, 00, 00, 0, time.Now().Location())

	tests := []struct {
		name                    string
		messagesToRepeat        []scheduledMessage[string, int, struct{}]
		scheduledItems          []scheduledMessage[string, int, struct{}]
		expectedScheduledKeys   []string
		expectedMaxSerialNumber uint64
	}{
		{
			name: `fresh serial number`,
			messagesToRepeat: []scheduledMessage[string, int, struct{}]{
				{
					Key:          `A`,
					SP:           0,
					serialNumber: 1,
					StartTime:    testNowTime,
				},
				{
					Key:          `A`,
					SP:           1,
					serialNumber: 2,
					StartTime:    testNowTime.Add(5 * time.Second),
				},
			},
			expectedScheduledKeys:   []string{`A`},
			expectedMaxSerialNumber: 2,
		},
		{
			name: `obsoleted serial number`,
			messagesToRepeat: []scheduledMessage[string, int, struct{}]{
				{
					Key:          `A`,
					SP:           0,
					serialNumber: 2,
					StartTime:    testNowTime,
				},
				{
					Key:          `A`,
					SP:           1,
					serialNumber: 1,
					StartTime:    testNowTime.Add(5 * time.Second),
				},
			},
			expectedScheduledKeys:   []string{`A`},
			expectedMaxSerialNumber: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initList := list.New()
			schedulerObj := newScheduler[string, int, struct{}](initList)

			for _, m := range test.messagesToRepeat {
				schedulerObj.OnRepeat(m, time.Now())
			}

			maxSerialNumber := uint64(0)
			resultKeys := make([]string, 0)
			for element := schedulerObj.scheduledItems.Front(); element != nil; element = element.Next() {
				m := element.Value.(scheduledMessage[string, int, struct{}])
				if m.serialNumber > maxSerialNumber {
					maxSerialNumber = m.serialNumber
				}
				resultKeys = append(resultKeys, m.Key)
			}

			require.Equal(t, test.expectedScheduledKeys, resultKeys)
			require.Equal(t, test.expectedMaxSerialNumber, maxSerialNumber)
		})
	}
}

func Test_SchedulerOnTimer(t *testing.T) {
	var testNowTime = time.Date(2023, 4, 20, 00, 00, 00, 0, time.Now().Location())

	t.Run(`empty scheduledItems`, func(t *testing.T) {
		dedupInCh := make(chan statefulMessage[string, int, struct{}], 10)

		schedulerObj := newScheduler[string, int, struct{}](nil)
		schedulerObj.OnTimer(dedupInCh, time.Now())
		schedulerObj.OnTimer(dedupInCh, time.Now())

		messagesToDedupIn := testMessagesReader(dedupInCh)

		// closing channels
		close(dedupInCh)

		require.Empty(t, messagesToDedupIn)
	})

	t.Run(`2 scheduled items`, func(t *testing.T) {
		dedupInCh := make(chan statefulMessage[string, int, struct{}], 10)

		scheduledItems := []scheduledMessage[string, int, struct{}]{
			{
				Key:          `A`,
				SP:           0,
				serialNumber: 1,
				StartTime:    testNowTime.Add(5 * time.Second),
			},
			{
				Key:          `B`,
				SP:           1,
				serialNumber: 2,
				StartTime:    testNowTime.Add(3 * time.Second),
			},
		}

		schedulerObj := newScheduler[string, int, struct{}](nil)
		// fulfilling ScheduledItems storage
		for _, m := range scheduledItems {
			schedulerObj.OnRepeat(m, testNowTime)
		}
		require.Equal(t, 3*time.Second, schedulerObj.lastDuration)

		schedulerObj.OnTimer(dedupInCh, testNowTime)
		require.Equal(t, 5*time.Second, schedulerObj.lastDuration)

		// closing channels
		close(dedupInCh)
	})

	t.Run(`dedupIn channel is busy`, func(t *testing.T) {
		dedupInCh := make(chan statefulMessage[string, int, struct{}], 1)

		scheduledItems := []scheduledMessage[string, int, struct{}]{
			{
				Key:          `A`,
				SP:           0,
				serialNumber: 1,
				StartTime:    testNowTime.Add(5 * time.Second),
			},
		}

		schedulerObj := newScheduler[string, int, struct{}](nil)
		// fulfilling ScheduledItems storage
		for _, m := range scheduledItems {
			schedulerObj.OnRepeat(m, testNowTime)
		}
		require.Equal(t, 5*time.Second, schedulerObj.lastDuration)

		// make dedupInCh busy
		dedupInCh <- statefulMessage[string, int, struct{}]{
			Key:          `B`,
			SP:           1,
			serialNumber: 2,
		}
		schedulerObj.OnTimer(dedupInCh, testNowTime)
		require.Equal(t, DedupInRetryInterval, schedulerObj.lastDuration)

		// closing channels
		close(dedupInCh)
	})
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
			name:          `2 keys are duplicated in 4 messages`,
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
			callerCh := make(chan statefulMessage[string, int, struct{}], 10)
			repeatCh := make(chan scheduledMessage[string, int, struct{}], 10)

			var messagesToCall []statefulMessage[string, int, struct{}]
			var messagesToRepeat []scheduledMessage[string, int, struct{}]
			var inProcessKeyCounter int
			go func() {
				testMessagesWriter(dedupInCh, test.messages)

				// closing channels
				close(dedupInCh)
				close(repeatCh)
			}()

			dedupIn(dedupInCh, callerCh, repeatCh, &InProcess, time.Now)

			messagesToCall = testMessagesReader(callerCh)
			messagesToRepeat = testMessagesReader(repeatCh)

			inProcessKeyCounter = 0
			InProcess.Range(func(_, _ any) bool {
				inProcessKeyCounter++
				return true
			})

			require.Len(t, messagesToCall, inProcessKeyCounter)
			require.Len(t, messagesToRepeat, 1)
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
			name:          `2 messages to report, 2 messages to repeat`,
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
					Key:          `C`,
					SP:           2,
					serialNumber: 1,
					StartTime:    nil,
					PV:           &pv,
				},
				{
					Key:          `D`,
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
			repeatCh := make(chan scheduledMessage[string, int, struct{}], 10)
			reporterCh := make(chan reportInfo[string, int], 10)

			var messagesToReport []reportInfo[string, int]
			var messagesToRepeat []scheduledMessage[string, int, struct{}]
			go func() {
				testMessagesWriter(repeaterCh, test.messages)

				close(repeaterCh)
			}()

			repeater(repeaterCh, repeatCh, reporterCh)

			messagesToReport = testMessagesReader(reporterCh)
			messagesToRepeat = testMessagesReader(repeatCh)

			require.Len(t, messagesToReport, 2)
			require.Len(t, messagesToRepeat, 2)
		})
	}
}

func testMessagesWriter[T any](ch chan<- T, arr []T) {
	for _, m := range arr {
		ch <- m
	}
}

func testMessagesReader[T any](ch <-chan T) []T {
	results := make([]T, 0)

	var val T
	ok := true
	for ok {
		select {
		case val, ok = <-ch:
			if ok {
				results = append(results, val)
			} else {
				return results
			}
		default:
			return results
		}
	}
	return results
}
