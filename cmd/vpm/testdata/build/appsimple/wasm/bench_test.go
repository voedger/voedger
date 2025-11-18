/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package main

import (
	"context"
	_ "embed"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/voedger/voedger/pkg/goutils/strconvu"
)

func Benchmark_Pbill(b *testing.B) {
	runBenchFunc(b, "Pbill", true)
	runBenchFunc(b, "Pbill", false)
}

func Benchmark_Sin(b *testing.B) {
	runBenchFunc(b, "Sin", true, 3, 4)
	runBenchFunc(b, "Sin", false, 3, 4)
}

func Benchmark_StupidPow(b *testing.B) {
	runBenchFunc(b, "StupidPow", true, 2, 200)
	runBenchFunc(b, "StupidPow", true, 2, 400)
	runBenchFunc(b, "StupidPow", false, 2, 200)
	runBenchFunc(b, "StupidPow", false, 2, 400)
}

func runBenchFunc(b *testing.B, fname string, compiler bool, args ...uint64) {
	benchName := fname
	if len(args) > 0 {
		benchName += "("
		for idx, arg := range args {
			if idx > 0 {
				benchName += ","
			}
			benchName += strconvu.UintToString(arg)
		}
		benchName += ")"
	}
	if compiler {
		benchName += "_Compiler"
	} else {
		benchName += "_Interpreter"
	}
	b.Run(benchName, func(b *testing.B) { benchFunc(b, fname, compiler, args...) })
}

func benchFunc(b *testing.B, fname string, compiler bool, args ...uint64) {
	ctx := context.Background()
	r, mod := newrm(b, extnogc(b), compiler)
	defer r.Close(ctx)

	f := mod.ExportedFunction(fname)

	for i := 0; i < b.N; i++ {
		callCtx, cancel := context.WithTimeout(ctx, time.Second*100)
		_, err := f.Call(callCtx, args...)
		if err != nil {
			b.Fatal(err)
		}
		cancel()
	}
}

func newrm(b require.TestingT, bytes []byte, compiler bool) (wazero.Runtime, api.Module) {
	// Choose the context to use for function calls.
	ctx := context.Background()

	var rtConf wazero.RuntimeConfig

	if compiler {
		rtConf = wazero.NewRuntimeConfigCompiler()
	} else {
		rtConf = wazero.NewRuntimeConfigInterpreter()
	}

	rtConf = rtConf.
		WithCoreFeatures(api.CoreFeatureSignExtensionOps | api.CoreFeatureBulkMemoryOperations).
		WithCloseOnContextDone(true).
		WithMemoryCapacityFromMax(true).
		WithMemoryLimitPages(50)

	r := wazero.NewRuntimeWithConfig(ctx, rtConf)

	{
		_, err := wasi_snapshot_preview1.Instantiate(ctx, r)
		require.NoError(b, err)
	}

	f := func() {}
	fhostGetKey := func(int32, int32, int32, int32) int64 { return 0 }
	fhostGetValue := func(int64) int64 { return 0 }
	fhostNewValue := func(int64) int64 { return 0 }
	hostRowWriterPutString := func(int64, int32, int32, int32, int32, int32) {}
	hostValueAsString := func(int64, int32, int32) int64 { return 0 }
	hostValueAsInt32 := func(int64, int32, int32) int32 { return 0 }
	hostRowWriterPutInt32 := func(id uint64, typ uint32, namePtr uint32, nameSize uint32, value int32) {}
	hostRowWriterPutInt64 := func(id uint64, typ uint32, namePtr uint32, nameSize uint32, value int64) {}
	hostValueAsInt64 := func(id uint64, namePtr uint32, nameSize uint32) (result uint64) { return 0 }
	hostValueAsValue := func(id uint64, namePtr uint32, nameSize uint32) (result uint64) { return 0 }
	hostValueGetAsValue := func(val uint64, index uint32) (result uint64) { return 0 }
	hostValueLength := func(id uint64) (result uint32) { return 0 }

	_, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(f).Export("callback").
		NewFunctionBuilder().WithFunc(fhostGetKey).Export("hostGetKey").
		NewFunctionBuilder().WithFunc(fhostGetValue).Export("hostGetValue").
		NewFunctionBuilder().WithFunc(fhostGetValue).Export("hostQueryValue").
		NewFunctionBuilder().WithFunc(fhostNewValue).Export("hostNewValue").
		NewFunctionBuilder().WithFunc(fhostNewValue).Export("hostNewValue").
		NewFunctionBuilder().WithFunc(hostRowWriterPutString).Export("hostRowWriterPutString").
		NewFunctionBuilder().WithFunc(hostValueAsString).Export("hostValueAsString").
		NewFunctionBuilder().WithFunc(hostValueAsInt32).Export("hostValueAsInt32").
		NewFunctionBuilder().WithFunc(hostRowWriterPutInt32).Export("hostRowWriterPutInt32").
		NewFunctionBuilder().WithFunc(hostRowWriterPutInt64).Export("hostRowWriterPutInt64").
		NewFunctionBuilder().WithFunc(hostValueAsInt64).Export("hostValueAsInt64").
		NewFunctionBuilder().WithFunc(hostValueAsValue).Export("hostValueAsValue").
		NewFunctionBuilder().WithFunc(hostValueGetAsValue).Export("hostValueGetAsValue").
		NewFunctionBuilder().WithFunc(hostValueLength).Export("hostValueLength").
		Instantiate(ctx)
	if err != nil {
		log.Panicln(err)
	}

	moduleCfg := wazero.NewModuleConfig().WithName("wasm").WithStdout(io.Discard).WithStderr(io.Discard)
	mod, err := r.InstantiateWithConfig(ctx, bytes, moduleCfg)

	require.NoError(b, err)
	return r, mod
}

var extnogcbytes []byte

func extnogc(t require.TestingT) []byte {
	if extnogcbytes != nil {
		return extnogcbytes
	}
	extnogcbytes, err := os.ReadFile("extnogc.wasm")
	if err != nil {
		// nolint error-is-as
		t.Errorf("error loading extnogc.wasm, use build.sh to make it. error: %s", err)
		t.FailNow()
	}
	return extnogcbytes
}
