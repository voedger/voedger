// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type noRelease struct{}

func (e noRelease) Release() {}

type workpiece struct {
	slots map[string]interface{}
}

func (w workpiece) Release() {
}

func TestBasicUsage_SyncPipeline(t *testing.T) {

	// Pipeline consists of few operators
	// Workpieces go through the pipelines operators

	// Sync pipeline can use any workpiece type, e.g.:
	// Simplified operator as a function, sets name slot
	funcOpSetName := func(_ context.Context, work IWorkpiece) error {
		work.(workpiece).slots["name"] = "michael"
		return nil
	}

	// Simplified operator as a function, sets age
	funcOpSetAge := func(ctx context.Context, work IWorkpiece) (err error) {
		work.(workpiece).slots["age"] = "39"
		return nil
	}

	// Operator which implements ISyncOperator: Prepare(), DoSync(), Close()
	// For each workpiece Prepare() is called first, then DoSync()
	// Close() is called once when pipeline shutdowns
	opSetCountry := mockSyncOp().
		doSync(func(ctx context.Context, work interface{}) (err error) {
			work.(workpiece).slots["country"] = "russia"
			return nil
		}).
		close(func() {}).
		create()

	// Wire pipeline with tree operators
	pipeline := NewSyncPipeline(context.Background(), "noname",
		WireFunc("Set name", funcOpSetName),
		WireFunc("Set age", funcOpSetAge),
		WireSyncOperator("Set country", opSetCountry),
	)
	defer pipeline.Close()

	work := workpiece{
		slots: make(map[string]interface{}),
	}

	// SendSync blocks until all operator finish
	err := pipeline.SendSync(work)
	require.NoError(t, err)

	// Test that operators have worked
	require.Equal(t, "michael", work.slots["name"])
	require.Equal(t, "39", work.slots["age"])
	require.Equal(t, "russia", work.slots["country"])
}

func TestBasicUsage_CatchInSyncOperator(t *testing.T) {

	// construct
	pipeline := NewSyncPipeline(context.Background(), "my-pipeline",
		WireFunc("apply-name", opName),
		WireFunc("fail-here", opError),
		WireSyncOperator("catch-error", mockSyncOp().
			doSync(func(ctx context.Context, work interface{}) (err error) {
				return nil
			}).
			catch(func(err error, work interface{}, context IWorkpieceContext) (newErr error) {
				work.(testwork).slots["error"] = err
				work.(testwork).slots["error-ctx"] = context
				return nil
			}).
			create()),
	)
	defer pipeline.Close()

	// do
	work := newTestWork()
	err := pipeline.SendSync(work)
	require.NoError(t, err)

	perr := work.slots["error"].(IErrorPipeline)

	require.Equal(t, "test failure", perr.Error())
	require.Equal(t, "fail-here", perr.GetOpName())
	require.Equal(t, "doSync", perr.GetPlace())
	pctx := work.slots["error-ctx"].(IWorkpieceContext)
	require.Equal(t, "my-pipeline", pctx.GetPipelineName())
	require.Equal(t, "operator: apply-name, operator: fail-here, operator: catch-error", pctx.GetPipelineStruct())
}

func TestBasicUsage_AsyncSwitchOperator(t *testing.T) {

	require := require.New(t)

	// Create AsyncPipeline with two AsyncPipeline's in AsyncSwitch branches and try to switch
	// Ref. design here https://dev.heeus.io/launchpad/#!13381

	// Branches will answer to this channel
	answers := make(chan int, 1)

	// Prepare branches

	branch1 := NewAsyncPipeline(context.Background(), "branch1",
		WireAsyncOperator("only-admins", mockAsyncOp().
			doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
				answers <- 1
				return
			}).
			create()),
	)
	branch2 := NewAsyncPipeline(context.Background(), "branch2",
		WireAsyncOperator("only-admins", mockAsyncOp().
			doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
				answers <- 2
				return
			}).
			create()),
	)

	// switch logic will use targetBranchName var

	var targetBranchName string

	switchLogic := mockSwitch{
		func(work interface{}) (branchName string, err error) { return targetBranchName, nil },
	}

	// AsyncSwitchOperator
	as := AsyncSwitchOperator(switchLogic, AsyncSwitchBranch("branch1", branch1), AsyncSwitchBranch("branch2", branch2))

	pipeline := NewAsyncPipeline(context.Background(), "AsyncPipeline with AsyncSwitch", WireAsyncOperator("AsyncSwitch", as))
	defer pipeline.Close()

	// Send to branch1

	targetBranchName = "branch1"
	require.NoError(pipeline.SendAsync(noRelease{}))
	require.Equal(1, <-answers)

	// Send to branch2

	targetBranchName = "branch2"
	require.NoError(pipeline.SendAsync(noRelease{}))
	require.Equal(2, <-answers)

	// Send in cycle

	const NumIters = 100

	for i := 0; i < NumIters; i++ {
		var branchNo int
		if i%2 == 1 {
			branchNo = 1
			targetBranchName = "branch1"
		} else {
			branchNo = 2
			targetBranchName = "branch2"
		}
		require.NoError(pipeline.SendAsync(noRelease{}))
		require.Equal(branchNo, <-answers)
	}

}

func TestBasicUsage_ForkOperator(t *testing.T) {
	operator := ForkOperator(
		func(work IWorkpiece, branchNumber int) (fork IWorkpiece, err error) {
			fork = newTestWork()
			for k, v := range work.(testwork).slots {
				fork.(testwork).slots[k] = v
			}
			return fork, nil
		},
		ForkBranch(mockSyncOp().
			doSync(func(ctx context.Context, work interface{}) (err error) {
				if work.(testwork).slots["name"] == "" {
					err = errors.New("name is empty")
				}
				return err
			}).create()),
		ForkBranch(mockSyncOp().
			doSync(func(ctx context.Context, work interface{}) (err error) {
				if work.(testwork).slots["age"].(int) < 18 {
					err = errors.New("too young")
				}
				return err
			}).create()))
	t.Run("Work should be valid", func(t *testing.T) {
		work := newTestWork()
		work.slots["name"] = "John"
		work.slots["age"] = 42

		err := operator.DoSync(context.Background(), work)

		require.NoError(t, err)
	})
	t.Run("Work should be invalid", func(t *testing.T) {
		work := newTestWork()
		work.slots["name"] = ""
		work.slots["age"] = 15

		err := operator.DoSync(context.Background(), work)

		var errErrInBranches ErrInBranches
		require.ErrorAs(t, err, &errErrInBranches)
		require.Len(t, errErrInBranches.Errors, 2)
	})
}

type workPiece struct {
	kind   string
	text   string
	number int
}

func (w *workPiece) Release() {}

func TestBasicUsage_SwitchOperator(t *testing.T) {
	switchLogic := mockSwitch{
		func(work interface{}) (branchName string, err error) {
			switch work.(*workPiece).kind {
			case "text":
				branchName = "splitter"
			case "number":
				branchName = "squarer"
			}
			return
		},
	}
	splitter := mockSyncOp().
		doSync(func(ctx context.Context, work interface{}) (err error) {
			w := work.(*workPiece)
			w.text = strings.Join(strings.Split(w.text, ""), " ")
			return nil
		}).
		create()
	squarer := mockSyncOp().
		doSync(func(ctx context.Context, work interface{}) (err error) {
			w := work.(*workPiece)
			w.number *= w.number
			return nil
		}).
		create()
	switchOperator := SwitchOperator(switchLogic,
		SwitchBranch("splitter", splitter),
		SwitchBranch("squarer", squarer))
	t.Run("Should execute splitter branch", func(t *testing.T) {
		work := workPiece{
			kind: "text",
			text: "Mazout",
		}

		_ = switchOperator.DoSync(context.Background(), &work)

		require.Equal(t, "M a z o u t", work.text)
	})
	t.Run("Should execute squarer branch", func(t *testing.T) {
		work := workPiece{
			kind:   "number",
			number: 9,
		}

		_ = switchOperator.DoSync(context.Background(), &work)

		require.Equal(t, 81, work.number)
	})
}

func TestBasicUsage_AsyncPipeline(t *testing.T) {
	// Async pipeline can use any workpiece that implements IWorkpiece interface, e.g.:
	type userEntry struct {
		noRelease
		id   int32
		name string
		role string
	}

	type usersGroupEntry struct {
		noRelease
		users []*userEntry
	}

	var nextIdentifier int32 = 1201
	var sortUsers = make([]*userEntry, 0)
	var aggrUsers = make([]*userEntry, 0)
	var result = make([]string, 0)

	funcOpFilterNoName := func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
		user := work.(*userEntry)
		if user.name == "" {
			return nil, nil
		}
		return work, nil
	}

	funcOpEnrichWithIds := func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
		user := work.(*userEntry)
		user.id = nextIdentifier
		nextIdentifier++
		return work, nil
	}

	// Async operator can accumulate workpieces and transmit further when calling the flush() method
	opSortByRoleAndName := mockAsyncOp().
		doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
			user := work.(*userEntry)
			sortUsers = append(sortUsers, user)
			return nil, nil
		}).
		flush(func(callback OpFuncFlush) (err error) {
			sort.SliceStable(sortUsers, func(i, j int) bool { return sortUsers[i].name < sortUsers[j].name })
			sort.SliceStable(sortUsers, func(i, j int) bool { return sortUsers[i].role < sortUsers[j].role })
			for _, user := range sortUsers {
				callback(user)
			}
			return nil
		}).
		create()

	opAggregateByRole := mockAsyncOp().
		doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
			user := work.(*userEntry)
			if len(aggrUsers) == 0 || user.role != aggrUsers[0].role {
				res := &usersGroupEntry{users: aggrUsers}
				if len(aggrUsers) == 0 {
					res = nil
				}
				aggrUsers = make([]*userEntry, 1)
				aggrUsers[0] = user
				return res, nil
			}
			aggrUsers = append(aggrUsers, user)
			return nil, nil
		}).
		flush(func(callback OpFuncFlush) (err error) {
			if len(aggrUsers) != 0 {
				callback(&usersGroupEntry{users: aggrUsers})
			}
			return nil
		}).
		create()

	funcOpResultWriter := func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
		usersGroup := work.(*usersGroupEntry)
		text := ""
		if usersGroup != nil {
			for _, user := range usersGroup.users {
				if text != "" {
					text += ", "
				}
				text += fmt.Sprintf("%d: %s", user.id, user.name)
			}
			result = append(result, fmt.Sprintf("[%s] %s", usersGroup.users[0].role, text))
		}
		return nil, nil
	}

	// Wire pipeline with async and sync operators
	pipeline := NewAsyncPipeline(context.Background(), "noname",
		WireAsyncOperator("filter-no-name", NewAsyncOp(funcOpFilterNoName)),
		WireAsyncOperator("enrich-with-ids", NewAsyncOp(funcOpEnrichWithIds)),
		WireAsyncOperator("sort-by-role-and-name", opSortByRoleAndName),
		WireAsyncOperator("aggregate-by-role", opAggregateByRole),
		WireAsyncOperator("writer", NewAsyncOp(funcOpResultWriter)),
	)

	require.NoError(t, pipeline.SendAsync(&userEntry{name: "Zeus", role: "client"}))
	require.NoError(t, pipeline.SendAsync(&userEntry{name: "Jack", role: "waiter"}))
	require.NoError(t, pipeline.SendAsync(&userEntry{name: "Bob", role: "client"}))
	require.NoError(t, pipeline.SendAsync(&userEntry{name: "Sam", role: "admin"}))
	require.NoError(t, pipeline.SendAsync(&userEntry{name: "Annie", role: "waiter"}))
	require.NoError(t, pipeline.SendAsync(&userEntry{role: "client"})) // no name
	require.NoError(t, pipeline.SendAsync(&userEntry{name: "John", role: "admin"}))
	require.NoError(t, pipeline.SendAsync(&userEntry{name: "Ron", role: "waiter"}))
	require.NoError(t, pipeline.SendAsync(&userEntry{name: "Peter", role: "admin"}))

	// Close blocks until all operator finish
	pipeline.Close()

	// Test that operators have worked
	require.Len(t, result, 3)
	require.Equal(t, "[admin] 1206: John, 1208: Peter, 1204: Sam", result[0])
	require.Equal(t, "[client] 1203: Bob, 1201: Zeus", result[1])
	require.Equal(t, "[waiter] 1205: Annie, 1202: Jack, 1207: Ron", result[2])

	// An additional test that the intermediate array is sorted correctly
	require.True(t, sort.SliceIsSorted(sortUsers, func(i, j int) bool {
		if sortUsers[i].role == sortUsers[j].role {
			return sortUsers[i].name < sortUsers[j].name
		}
		return sortUsers[i].role < sortUsers[j].role
	}))
}

func TestBasicUsage_AsyncPipeline_FlushByTimer(t *testing.T) {
	t.Run("Should flush with valid workpieces", func(t *testing.T) {
		mustHave := func(result *sync.Map, key string) bool {
			_, ok := result.Load(key)
			for !ok {
				time.Sleep(time.Duration(10) * time.Millisecond)
				_, ok = result.Load(key)
			}
			return ok
		}
		times := 0
		records := make([]string, 0)
		result := new(sync.Map)
		pipeline := NewAsyncPipeline(
			context.Background(),
			"async pipeline",
			WireAsyncOperator("filter waiters", mockAsyncOp().
				doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					if work.(userEntry).role == "waiter" {
						return nil, nil
					}
					return work, nil
				}).create()),
			WireAsyncOperator("aggregate names", mockAsyncOp().
				doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					records = append(records, strings.ToUpper(work.(userEntry).name))
					return nil, nil
				}).
				flush(func(callback OpFuncFlush) (err error) {
					times++
					for _, record := range records {
						result.Store(record, true)
					}
					return nil
				}).create(), time.Duration(20)*time.Millisecond))

		require.NoError(t, pipeline.SendAsync(userEntry{role: "admin", name: "John"}))
		require.True(t, mustHave(result, "JOHN"))
		require.NoError(t, pipeline.SendAsync(userEntry{role: "admin", name: "Chip"}))
		require.NoError(t, pipeline.SendAsync(userEntry{role: "waiter", name: "Wrong man"}))
		require.NoError(t, pipeline.SendAsync(userEntry{role: "admin", name: "Dale"}))
		require.True(t, mustHave(result, "CHIP"))
		require.True(t, mustHave(result, "DALE"))
		require.GreaterOrEqual(t, times, 1)
	})
	t.Run("Should no flush without workpiece", func(t *testing.T) {
		times := 0
		pipeline := NewAsyncPipeline(
			context.Background(),
			"async pipeline",
			WireAsyncOperator("filter waiters", mockAsyncOp().
				doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					if work.(userEntry).role == "waiter" {
						return nil, nil
					}
					return work, nil
				}).create()),
			WireAsyncOperator("aggregate names", mockAsyncOp().
				doAsync(func(ctx context.Context, work IWorkpiece) (outWork IWorkpiece, err error) {
					return nil, err
				}).
				flush(func(callback OpFuncFlush) (err error) {
					times++
					return nil
				}).create(), time.Duration(20)*time.Millisecond))

		require.NoError(t, pipeline.SendAsync(userEntry{role: "waiter"}))
		time.Sleep(time.Duration(50) * time.Millisecond)
		require.NoError(t, pipeline.SendAsync(userEntry{role: "waiter"}))
		time.Sleep(time.Duration(50) * time.Millisecond)
		require.NoError(t, pipeline.SendAsync(userEntry{role: "waiter"}))
		time.Sleep(time.Duration(50) * time.Millisecond)

		require.Equal(t, 0, times)
	})
}

func TestBasicUsage_ServiceOperator(t *testing.T) {

	// In this test we will prepare, start and stop two services

	require := require.New(t)

	// We will keep track of stopped services numbers here
	stopLog := make(chan int, 2)

	// Track IService.Prepare() handle
	prepareLog := make(chan int, 2)

	// Need context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Prepare service1
	// Service must implement Prepare(), Run() and Stop() methods of IService:
	// Stop() will be called after ctx is cancelled, we will use ctx so Stop() implementation is not necessary
	service1 := &mockIService{
		prepare: func(work interface{}) error {
			prepareLog <- 1
			return nil
		},
		run: func(ctx context.Context) {
			<-ctx.Done()
			log.Println("service1 done")
			stopLog <- 1
		},
		stop: func() {},
	}

	service2 := &mockIService{
		prepare: func(work interface{}) error {
			prepareLog <- 2
			return nil
		},
		run: func(ctx context.Context) {
			<-ctx.Done()
			stopLog <- 2
			log.Println("service2 done")

		},
		stop: func() {},
	}

	// Wire SyncPipeline with two services

	p := NewSyncPipeline(ctx, "testPipeline",
		WireSyncOperator("service1", ServiceOperator(service1)),
		WireSyncOperator("service2", ServiceOperator(service2)),
	)

	workpiece := &noRelease{}

	// Start services
	// NB: Only one SendSync is allowed
	require.NoError(p.SendSync(workpiece))

	select {
	case prep := <-prepareLog:
		require.Equal(1, prep)
	default:
		t.Fatal()
	}
	select {
	case prep := <-prepareLog:
		require.Equal(2, prep)
	default:
		t.Fatal()
	}

	// Must cancel() context before Close()
	cancel()

	p.Close()

	// Check that two services stopped (in random order)

	<-stopLog
	<-stopLog

}
