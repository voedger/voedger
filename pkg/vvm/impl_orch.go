/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Alisher Nurmanov
 */

package vvm

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/ielections"
	"github.com/voedger/voedger/pkg/vvm/storage"
)

// [~server.design.orch/VVM.LaunchVVM~impl]
func (vvm *VoedgerVM) Launch(leadershipDurationSeconds ielections.LeadershipDurationSeconds, leadershipAcquisitionDuration LeadershipAcquisitionDuration) context.Context {
	if vvm.leadershipCtx != nil {
		panic("VVM is launched already")
	}
	go vvm.shutdowner()
	err := vvm.tryToAcquireLeadership(leadershipDurationSeconds, leadershipAcquisitionDuration)
	if err == nil {
		if err = vvm.ServicePipeline.SendSync(ignition{}); err != nil {
			err = errors.Join(err, ErrVVMServicesLaunch)
		}
	}

	if err != nil {
		vvm.updateProblem(err)
	}

	return vvm.problemCtx
}

// [~server.design.orch/VVM.Shutdown~impl]
func (vvm *VoedgerVM) Shutdown() error {
	if vvm.leadershipCtx == nil {
		select {
		case <-vvm.problemCtx.Done():
		default:
			panic("VVM must be launched before shutdown")
		}
	}
	// Ensure we only close the vvmShutCtx once
	vvm.vvmShutCtxCancel()

	// Block until everything is fully shutdown
	<-vvm.shutdownedCtx.Done()

	// additionally close problemCtx for the case when we call vvm.Shutdown when problemCtx is not closed yet to avoid context leak
	vvm.problemCtxCancel()

	vvm.leadershipCtx = nil
	select {
	case err := <-vvm.problemErrCh:
		return err
	default:
		return nil
	}
}

// [~server.design.orch/VVM.Shutdowner~impl]
func (vvm *VoedgerVM) shutdowner() {
	// Wait for VVM.vvmShutCtx
	<-vvm.vvmShutCtx.Done()

	// Shutdown everything but LeadershipMonitor and elections
	{
		// close VVM.servicesShutCtx
		vvm.servicesShutCtxCancel()

		// wait for services to stop
		vvm.vvmCtxCancel()
		vvm.ServicePipeline.Close()
		vvm.vvmCleanup()
	}

	// Shutdown LeadershipMonitor (close VVM.monitorShutCtx and wait for LeadershipMonitor to stop)
	vvm.monitorShutCtxCancel()
	vvm.monitorShutWg.Wait()

	// Cleanup elections (leadership is released here)
	vvm.electionsCleanup()

	// Close VVM.shutdownedCtx
	vvm.shutdownedCtxCancel()
}

// leadershipMonitor is a routine that monitors the leadership context.
// [~server.design.orch/LeadershipMonitor~impl]
func (vvm *VoedgerVM) leadershipMonitor(leadershipDurationSeconds ielections.LeadershipDurationSeconds) {
	defer vvm.monitorShutWg.Done()

	select {
	case <-vvm.leadershipCtx.Done():
		// leadership is lost
		go vvm.killerRoutine(leadershipDurationSeconds)
		vvm.updateProblem(ErrLeadershipLost)
	case <-vvm.monitorShutCtx.Done():
	}
}

// killerRoutine is a routine that kills the VVM process after a quarter of the leadership duration
func (vvm *VoedgerVM) killerRoutine(leadershipDurationSeconds ielections.LeadershipDurationSeconds) {
	// [~server.design.orch/processKillThreshold~impl]
	// nolint:revive
	processKillThreshold := time.Duration(leadershipDurationSeconds) * time.Second / 4
	time.Sleep(processKillThreshold)
	logger.Error("the process is still alive after the time alloted for graceful shutdown -> terminating...")
	os.Exit(1)
}

// tryToAcquireLeadership tries to acquire leadership in loop
// [~server.design.orch/VVM.tryToAcquireLeadership~impl]
func (vvm *VoedgerVM) tryToAcquireLeadership(leadershipDurationSeconds ielections.LeadershipDurationSeconds,
	leadershipAcquisitionDuration LeadershipAcquisitionDuration) error {
	elections, electionsCleanup := ielections.Provide(vvm.TTLStorage, vvm.ITime)
	vvm.electionsCleanup = electionsCleanup

	leadershipAcquistionTimerCh := vvm.ITime.NewTimerChan(time.Duration(leadershipAcquisitionDuration))

	// to inform the test that the leadership acquisition has started
	vvm.leadershipAcquisitionTimerArmed <- struct{}{}

	vvmIdx := storage.TTLStorageImplKey(1)
	leadershipAcquisitionDeadline := vvm.ITime.Now().Add(time.Duration(leadershipAcquisitionDuration))
	for vvm.ITime.Now().Before(leadershipAcquisitionDeadline) {
		select {
		case <-leadershipAcquistionTimerCh:
			return ErrVVMLeadershipAcquisition
		default:
			// Try to acquire leadership
			vvm.leadershipCtx = elections.AcquireLeadership(vvmIdx, vvm.ip.String(), leadershipDurationSeconds)
			if vvm.leadershipCtx != nil {
				// If leadership is acquired
				vvm.monitorShutWg.Add(1)
				go vvm.leadershipMonitor(leadershipDurationSeconds)
				return nil
			}
			// Try the next VVM index in the cluster from 1 to numVVM
			vvmIdx++
			if vvmIdx > vvm.numVVM {
				vvmIdx = 1
			}
			time.Sleep(time.Second)
		}
	}
	// notest
	return ErrVVMLeadershipAcquisition
}

// updateProblem writes a critical error into problemErrCh exactly once
// and sets the cause on problemCtx. This ensures no double-writes of errors.
// [~server.design.orch/VVM.updateProblem~impl]
func (vvm *VoedgerVM) updateProblem(err error) {
	// The sync.Once ensures we only do this logic once
	vvm.problemCtxCancel()

	vvm.problemCtxErrOnce.Do(func() {
		vvm.problemErrCh <- err
	})
}
