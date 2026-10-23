/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 */

package vit

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/istorage/provider"
)

// RestartVVM synchronously replaces the VVM over the same storage, without
// advancing mock time or provisioning configured entities. A successful return
// means the replacement is running and ready for requests. The VIT stays acquired.
// Calls must be exclusive: do not use VIT concurrently or from a stopped hook.
// Server restart is not supported in benchmarks or stress tests.
func (vit *VIT) RestartVVM(opts ...RestartOptFunc) (err error) {
	vit.T.Helper()
	if _, benchmark := vit.T.(*testing.B); benchmark {
		return errors.New("RestartVVM is not supported in benchmarks")
	}
	if err := vit.serverUnavailable(); err != nil {
		return err
	}
	options := restartOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	cfg := options.config
	if cfg == nil {
		cfg = vit.effectiveConfig
	}
	completed := false
	defer vit.cleanupOnFailure(&completed)

	if err := vit.cleanupOnRestart(); err != nil {
		return fmt.Errorf("stop VVM for restart: %w", err)
	}
	if options.afterStop != nil {
		options.afterStop(restartStorage{provider: provider.Provide(vit.storageFactory, vit.storageSuffix)})
	}
	if _, err := vit.startVVM(cfg); err != nil {
		return err
	}
	vit.RefreshTokens()
	vit.effectiveConfig = cfg
	completed = true
	return nil
}

// WithAfterStop runs the callback after the old VVM stops and before its replacement starts.
// The callback must use only the supplied storage, without calling VIT operations.
func WithAfterStop(callback func(RestartStorage)) RestartOptFunc {
	return func(options *restartOptions) {
		options.afterStop = callback
	}
}

// WithNewConfig replaces the effective configuration while retaining storage and ownership.
// A nil configuration reuses the current effective configuration.
func WithNewConfig(config *VITConfig) RestartOptFunc {
	return func(options *restartOptions) {
		options.config = config
	}
}

// Deferred cleanup leaves the original error, panic or Goexit with the caller.
func (vit *VIT) cleanupOnFailure(completed *bool) {
	if !*completed {
		vit.cleanupServer()
	}
}

func (vit *VIT) requireRunning() {
	vit.T.Helper()
	if err := vit.serverUnavailable(); err != nil {
		vit.T.Fatal(err)
	}
}

func (vit *VIT) serverUnavailable() error {
	if vit.isFinalized || vit.VoedgerVM == nil || vit.serverClosed {
		return errors.New("VIT must be acquired and have a running server")
	}
	return nil
}

// cleanupOwnVIT ends the test and stops its current server.
func (vit *VIT) cleanupOwnVIT() {
	vit.T.Helper()
	defer vit.cleanupServer()
	vit.runConfigCleanups()
}

// cleanupSharedVIT ends the test while keeping its current server reusable.
func (vit *VIT) cleanupSharedVIT() {
	vit.T.Helper()
	defer func() {
		if vit.serverClosed {
			vit.cleanupServer()
			return
		}
		vit.releaseResources()
		vit.emailCaptor.checkEmpty(vit.T)
		vit.checkVVMProblemCtx()
	}()
	vit.runConfigCleanups()
}

// cleanupOnRestart releases runtime resources and stops the server without
// consuming config cleanups or ending the current acquisition.
func (vit *VIT) cleanupOnRestart() error {
	return vit.shutdownVVM()
}

// Consume only the current test's snapshot; retain configured definitions for reuse.
func (vit *VIT) runConfigCleanups() {
	callbacks := vit.currentAfterTest
	vit.currentAfterTest = nil
	for _, callback := range slices.Backward(callbacks) {
		defer callback(vit)
	}
}

// releaseResources disposes of resources that cannot survive restart or teardown.
func (vit *VIT) releaseResources() {
	vit.lock.Lock()
	cleanups := vit.currentTestCleanups
	vit.currentTestCleanups = nil
	vit.lock.Unlock()

	// The client is reused by later shared acquisitions; close idle connections
	// on every release without retaining a separate cleanup field.
	if vit.httpClient != nil {
		cleanups = append(cleanups, vit.httpClient.CloseIdleConnections)
	}
	// Run all callbacks in registration order, even on panic or Goexit.
	for _, cleanup := range slices.Backward(cleanups) {
		defer cleanup()
	}
}

func (vit *VIT) closeBLOBReader(reader iblobstorage.BLOBReader) {
	vit.T.Helper()
	if reader.ReadCloser == nil {
		return
	}
	defer reader.Close()
	buf := make([]byte, 1)
	_, err := reader.Read(buf)
	if errors.Is(err, io.EOF) {
		return
	}
	require.NoError(vit.T, err)
	_, err = io.Copy(io.Discard, reader)
	require.NoError(vit.T, err)
	vit.T.Fatal("BLOB reader is not read out")
}

func (vit *VIT) shutdownVVM() (err error) {
	if vit.serverClosed {
		return nil
	}
	defer func() {
		if vit.emailCaptor != nil {
			vit.emailCaptor.checkEmpty(vit.T)
			vit.emailCaptor.shutDown()
		}
		vit.serverClosed = true
	}()
	defer func() {
		if vit.vvmProblemCtx != nil {
			err = vit.VoedgerVM.Shutdown()
		}
	}()
	vit.releaseResources()
	return nil
}

func (vit *VIT) cleanupServer() {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			vit.T.Errorf("VIT server cleanup panicked: %v", panicValue)
		}
	}()
	if err := vit.shutdownVVM(); err != nil {
		// Preserve the original restart error or panic; reporting cleanup must
		// not terminate its deferred failure handling with another FailNow.
		vit.T.Errorf("VIT server cleanup: %v", err)
	}
}
