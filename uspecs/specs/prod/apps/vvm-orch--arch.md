# Context subsystem architecture: VVM Orchestration

## Problem statement

Design a reliable orchestration mechanism for VVM that ensures:

- VVM goroutines work only if leadership is acquired and held
- Clean termination of all goroutines
- Concurrent-safe error handling
- Graceful shutdown capabilities

## Overview

VVM orchestration manages the lifecycle of VVM goroutines through a leadership-based coordination mechanism. The system ensures that application services run only when VVM holds leadership, and provides graceful shutdown with automatic process termination if leadership is lost.

Key mechanisms:

- Leadership acquisition: VVM acquires leadership by writing a TTL record (default: 20s) to storage before starting services, with acquisition timeout of 120s
- Active renewal: Elections component renews leadership every TTL/2 interval (default: 10s) via CompareAndSwap operations
- Passive monitoring: leadershipMonitor goroutine waits for leadership loss notification via context cancellation
- Automatic termination: killerRoutine forcefully terminates the process (default: 5s after leadership loss) if graceful shutdown fails
- Sequential shutdown: VVM.Shutdown() terminates goroutines in order, waiting for each to exit before proceeding

The orchestration uses context cancellation for signaling and WaitGroups for synchronization, ensuring all goroutines terminate cleanly during shutdown.

## Goroutines

### Goroutine hierarchy

```text
VVMHost
- Launcher
  - LeadershipMonitor
    - killerRoutine
  - ServicePipeline
- Shutdowner
```

### Goroutine launch flow

`VVM.Launch()` spawns goroutines in order:

1. VVMHost calls VVM.Launch()
   - VVM.Launch() runs on main thread (caller's thread)

2. VVM.Launch() spawns **Launcher** goroutine
   - VVM calls go Launcher()
   - Launcher goroutine starts
   - VVM.Launch() returns immediately to VVMHost
     - Does not wait for Launcher to complete
     - Launcher runs concurrently in background

3. **Launcher** acquires leadership
   - Launcher calls elections.AcquireLeadership()
     - Blocks here until leadership is acquired
     - Elections writes leadership record to TTL storage
     - Elections spawns **maintainLeadership** goroutine
       - Runs timer at TTL/2 interval (e.g., every 2.5s for 5s TTL)
       - Actively renews leadership by calling CompareAndSwap()
       - If renewal fails, cancels `leadershipCtx` and exits
     - Returns leadershipCtx when leadership is confirmed

4. Launcher spawns **leadershipMonitor** goroutine
   - Launcher calls go leadershipMonitor(leadershipCtx)
   - leadershipMonitor goroutine starts
   - Waits passively on `leadershipCtx`.Done()
     - Wakes up when maintainLeadership cancels leadershipCtx
     - Spawns killerRoutine on leadership loss
   - Launcher continues (does not wait)

5. Launcher spawns **ServicePipeline** goroutine
   - Launcher calls go ServicePipeline()
   - ServicePipeline goroutine starts
   - Runs application services
   - Launcher continues (does not wait)

6. Launcher waits for shutdown signal
   - Launcher blocks on vvmShutCtx.Done()
   - Remains blocked until VVM.Shutdown() is called

7. **maintainLeadership** detects leadership loss (conditional)
   - Runs continuously, renewing leadership every TTL/2 interval
   - If renewal fails (storage error, record gone, network partition, etc.)
     - maintainLeadership calls releaseLeadership()
     - releaseLeadership cancels leadershipCtx
     - leadershipMonitor wakes up on leadershipCtx.Done()
     - leadershipMonitor spawns killerRoutine
     - killerRoutine waits for delay period
     - killerRoutine forcefully terminates process (os.Exit)
   - If renewal succeeds
     - maintainLeadership continues renewing

### Shutdown sequence

`VVM.Shutdown()` terminates goroutines in order:

1. VVM cancels vvmShutCtx
   - Global shutdown signal sent to all components

2. VVM terminates **leadershipMonitor**
   - VVM calls cancel(monitorShutCtx)
   - VVM calls monitorShutWg.Wait()
     - Blocks here, waiting for leadershipMonitor to finish
     - leadershipMonitor detects monitorShutCtx.Done()
     - leadershipMonitor calls monitorShutWg.Done()
     - Wait() unblocks
     - leadershipMonitor goroutine exits

3. VVM terminates **ServicePipeline**
   - VVM calls cancel(servicesShutCtx)
   - VVM calls servicesShutWg.Wait()
     - Blocks here, waiting for ServicePipeline to finish
     - ServicePipeline detects servicesShutCtx.Done()
     - ServicePipeline performs cleanup
     - ServicePipeline calls servicesShutWg.Done()
     - Wait() unblocks
     - ServicePipeline goroutine exits

4. VVM cleans up elections
   - VVM calls electionsCleanup()
     - Elections sets isFinalized flag (prevents new acquisitions)
     - Elections calls releaseLeadership() for all active leaderships
       - Calls CompareAndDelete() to remove leadership record from storage
       - Cancels leadershipCtx (signals maintainLeadership to stop)
       - Waits for maintainLeadership goroutine to exit
     - Returns when all maintainLeadership goroutines have exited

5. VVM waits for **Launcher**
   - VVM calls launcherShutWg.Wait()
     - Blocks here, waiting for Launcher to finish
     - Launcher detects shutdown (via vvmShutCtx)
     - Launcher performs cleanup
     - Launcher calls launcherShutWg.Done()
     - Wait() unblocks
     - Launcher goroutine exits

6. VVM signals shutdown complete
   - VVM calls cancel(shutdownedCtx)
   - Signals to any waiters that shutdown is complete
   - VVM returns to VVMHost (nil or error)

VVM terminates all goroutines cleanly except killerRoutine (if it was spawned due to leadership loss, it will forcefully terminate the process after its delay).

### Key constants

Orchestration timing constants:

- **DefaultLeadershipDurationSeconds** = 20 seconds
  - Location: [pkg/vvm/consts.go](../../../../pkg/vvm/consts.go)
  - TTL duration for leadership record in storage
  - Used by elections component to set TTL on leadership records

- **Leadership renewal interval** = LeadershipDurationSeconds / 2
  - Location: [pkg/ielections/impl.go](../../../../pkg/ielections/impl.go) (line 58)
  - Calculated dynamically: `tickerInterval := time.Duration(ttlSeconds) * time.Second / 2`
  - maintainLeadership goroutine renews leadership at this interval
  - Example: For 20s TTL, renewal happens every 10s

- **processKillThreshold** = LeadershipDurationSeconds / 4
  - Location: [pkg/vvm/impl_orch.go](../../../../pkg/vvm/impl_orch.go) (line 111)
  - Calculated dynamically: `time.Duration(leadershipDurationSeconds) * time.Second / 4`
  - killerRoutine waits this long before forcefully terminating process
  - Example: For 20s TTL, process is killed after 5s if still alive

- **DefaultLeadershipAcquisitionDuration** = 120 seconds
  - Location: [pkg/vvm/consts.go](../../../../pkg/vvm/consts.go)
  - Maximum time to wait for leadership acquisition during VVM.Launch()
  - Launcher tries to acquire leadership in a loop until this timeout

Timing relationships:

```text
LeadershipDurationSeconds = 20s (default)
  |
  +-> Renewal interval = 20s / 2 = 10s (maintainLeadership renews every 10s)
  |
  +-> Kill threshold = 20s / 4 = 5s (killerRoutine waits 5s before os.Exit)

Leadership acquisition timeout = 120s (Launcher tries for up to 2 minutes)
```

---

## Key components

Subsystem packages:

- [pkg/vvm](../../../../pkg/vvm)
  - Provides VVM lifecycle orchestration and ownership of shutdown sequencing

- [pkg/vvm/impl_orch.go](../../../../pkg/vvm/impl_orch.go)
  - Leadership acquisition, leadership monitor, shutdowner, error propagation

- [pkg/vvm/impl_orch_test.go](../../../../pkg/vvm/impl_orch_test.go)
  - Lifecycle, shutdown, and leadership behavior tests

- [pkg/ielections](../../../../pkg/ielections)
  - Leadership acquisition contracts and implementation

- [pkg/vvm/storage](../../../../pkg/vvm/storage)
  - ITTLStorage implementation backed by sysvvm keyspace

### Manual testing research

- airs-bp3/rsch/20250226-orch

Flow

- scylla.sh
  - Start scylla
- bp3_1.sh
  - Start the first bp3 instance, it takes the leadership
  - docker pull untillpro/airs-bp:alpha
- bp3_2.sh
  - Start the second bp3 instance, it waits for the leadership
- bp3_1_stop.sh
  - bp3_1 stops
  - bp3_2 takes the leadership
- bp3_1.sh
  - bp3_1 waits for the leadership
- bp3_2_stop.sh
  - bp3_2 stops
  - bp3_1 takes the leadership

### References

- [Original VVM Orchestration Design](https://github.com/voedger/voedger-internals/blob/4a5957e0e97917da1788cf1a3426187510dc875e/docs/server/design/orch.md)
