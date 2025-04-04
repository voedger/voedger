/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/

// nolint G115
package iextenginewazero

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/iextengine"
	imetrics "github.com/voedger/voedger/pkg/metrics"
	"github.com/voedger/voedger/pkg/processors"
	safe "github.com/voedger/voedger/pkg/state/isafestateapi"
	"github.com/voedger/voedger/pkg/state/safestate"
)

type limitedWriter struct {
	limit int
	buf   []byte
}

type wazeroExtPkg struct {
	moduleCfg wazero.ModuleConfig
	compiled  wazero.CompiledModule
	module    api.Module
	exts      map[string]api.Function
	wasmData  []byte

	funcMalloc api.Function
	funcFree   api.Function

	funcVer          api.Function
	funcGetHeapInuse api.Function
	funcGetHeapSys   api.Function
	funcGetMallocs   api.Function
	funcGetFrees     api.Function
	funcGc           api.Function
	funcOnReadValue  api.Function

	allocatedBufs []*allocatedBuf
	stdout        limitedWriter
	pkgName       string
	//	memBackup     *api.MemoryBackup
	//	recoverMem    []byte
}

type wazeroExtEngine struct {
	app appdef.AppQName

	compile            bool
	invocationsTotal   *imetrics.MetricValue
	invocationsSeconds *imetrics.MetricValue
	errorsTotal        *imetrics.MetricValue
	recoversTotal      *imetrics.MetricValue
	config             *iextengine.ExtEngineConfig
	modules            map[string]*wazeroExtPkg
	host               api.Module
	rtm                wazero.Runtime

	wasiCloser api.Closer

	// Invoke-related!
	safeApi safe.IStateSafeAPI

	ctx         context.Context
	pkg         *wazeroExtPkg
	autoRecover bool
}

type allocatedBuf struct {
	addr uint32
	offs uint32
	cap  uint32
}

type extensionEngineFactory struct {
	wasmConfig iextengine.WASMFactoryConfig
	vvmName    processors.VVMName
	imetrics   imetrics.IMetrics
}

func newLimitedWriter(limit int) limitedWriter {
	return limitedWriter{limit: limit}
}

func (w *limitedWriter) Reset() {
	w.buf = w.buf[:0]
}

func (w *limitedWriter) Write(p []byte) (n int, err error) {
	if len(w.buf)+len(p) > w.limit {
		w.buf = append(w.buf, p[:w.limit-len(w.buf)]...)
	} else {
		w.buf = append(w.buf, p...)
	}
	return len(p), nil
}

func (f extensionEngineFactory) New(ctx context.Context, app appdef.AppQName, packages []iextengine.ExtensionModule, config *iextengine.ExtEngineConfig, numEngines uint) (engines []iextengine.IExtensionEngine, err error) {
	for i := uint(0); i < numEngines; i++ {
		engine := &wazeroExtEngine{
			app:                app,
			modules:            make(map[string]*wazeroExtPkg),
			config:             config,
			compile:            f.wasmConfig.Compile,
			invocationsTotal:   f.imetrics.AppMetricAddr(metric_voedger_pee_invocations_total, string(f.vvmName), app),
			invocationsSeconds: f.imetrics.AppMetricAddr(metric_voedger_pee_invocations_seconds, string(f.vvmName), app),
			errorsTotal:        f.imetrics.AppMetricAddr(metric_voedger_pee_errors_total, string(f.vvmName), app),
			recoversTotal:      f.imetrics.AppMetricAddr(metric_voedger_pee_recovers_total, string(f.vvmName), app),
			autoRecover:        true,
		}
		err = engine.init(ctx)
		if err != nil {
			return engines, err
		}
		engines = append(engines, engine)
	}

	for _, pkg := range packages {
		if pkg.ModuleURL.Scheme == "file" && (pkg.ModuleURL.Host == "" || strings.EqualFold("localhost", pkg.ModuleURL.Scheme)) {
			path := pkg.ModuleURL.Path
			if runtime.GOOS == "windows" {
				path = strings.TrimPrefix(path, "/")
			}

			wasmdata, err := os.ReadFile(path)

			if err != nil {
				return nil, err
			}

			for _, eng := range engines {
				err = eng.(*wazeroExtEngine).initModule(ctx, pkg.Path, wasmdata, pkg.ExtensionNames)
				if err != nil {
					return nil, err
				}
			}
		} else {
			return nil, errors.New("unsupported URL: " + pkg.ModuleURL.String())
		}
	}
	return engines, nil
}

func (f *wazeroExtEngine) SetLimits(limits iextengine.ExtensionLimits) {
	// f.cep.Duration = limits.ExecutionInterval
}

func (f *wazeroExtPkg) importFuncs(funcs map[string]*api.Function) error {

	for k, v := range funcs {
		*v = f.module.ExportedFunction(k)
		if *v == nil {
			return fmt.Errorf("missing exported function: %s", k)
		}
	}
	return nil
}

func (f *wazeroExtEngine) init(ctx context.Context) error {
	var err error
	var memPages = f.config.MemoryLimitPages
	if memPages == 0 {
		memPages = iextengine.DefaultMemoryLimitPages
	}
	if memPages > maxMemoryPages {
		return errors.New("maximum allowed MemoryLimitPages is 0xffff")
	}
	// Total amount of memory must be at least 170% of WasmPreallocatedBufferSize
	const memoryLimitCoef = 1.7
	memoryLimit := memPages * iextengine.MemoryPageSize
	limit := math.Trunc(float64(WasmPreallocatedBufferSize) * float64(memoryLimitCoef))
	if uint32(memoryLimit) <= uint32(limit) { // nolint G115 memoryLimit is max maxMemoryPages, limit is WasmPreallocatedBufferSize*1.7
		return fmt.Errorf("the minimum limit of memory is: %.1f bytes, requested limit is: %.1f", limit, float32(memoryLimit))
	}

	var rtConf wazero.RuntimeConfig

	if f.compile {
		rtConf = wazero.NewRuntimeConfigCompiler()
	} else {
		rtConf = wazero.NewRuntimeConfigInterpreter()
	}
	rtConf = rtConf.
		WithCoreFeatures(api.CoreFeatureBulkMemoryOperations | api.CoreFeatureSignExtensionOps | api.CoreFeatureNonTrappingFloatToIntConversion).
		WithCloseOnContextDone(true).
		WithMemoryCapacityFromMax(true).
		WithMemoryLimitPages(uint32(memPages))

	f.rtm = wazero.NewRuntimeWithConfig(ctx, rtConf)
	f.wasiCloser, err = wasi_snapshot_preview1.Instantiate(ctx, f.rtm)

	if err != nil {
		return err
	}

	f.host, err = f.rtm.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(f.hostGetKey).Export("hostGetKey").
		NewFunctionBuilder().WithFunc(f.hostMustExist).Export("hostGetValue").
		NewFunctionBuilder().WithFunc(f.hostCanExist).Export("hostQueryValue").
		NewFunctionBuilder().WithFunc(f.hostReadValues).Export("hostReadValues").
		// IKey
		NewFunctionBuilder().WithFunc(f.hostKeyAsString).Export("hostKeyAsString").
		NewFunctionBuilder().WithFunc(f.hostKeyAsBytes).Export("hostKeyAsBytes").
		NewFunctionBuilder().WithFunc(f.hostKeyAsInt32).Export("hostKeyAsInt32").
		NewFunctionBuilder().WithFunc(f.hostKeyAsInt64).Export("hostKeyAsInt64").
		NewFunctionBuilder().WithFunc(f.hostKeyAsFloat32).Export("hostKeyAsFloat32").
		NewFunctionBuilder().WithFunc(f.hostKeyAsFloat64).Export("hostKeyAsFloat64").
		NewFunctionBuilder().WithFunc(f.hostKeyAsBool).Export("hostKeyAsBool").
		NewFunctionBuilder().WithFunc(f.hostKeyAsQNamePkg).Export("hostKeyAsQNamePkg").
		NewFunctionBuilder().WithFunc(f.hostKeyAsQNameEntity).Export("hostKeyAsQNameEntity").
		// IValue
		NewFunctionBuilder().WithFunc(f.hostValueLength).Export("hostValueLength").
		NewFunctionBuilder().WithFunc(f.hostValueAsValue).Export("hostValueAsValue").
		NewFunctionBuilder().WithFunc(f.hostValueAsString).Export("hostValueAsString").
		NewFunctionBuilder().WithFunc(f.hostValueAsBytes).Export("hostValueAsBytes").
		NewFunctionBuilder().WithFunc(f.hostValueAsInt32).Export("hostValueAsInt32").
		NewFunctionBuilder().WithFunc(f.hostValueAsInt64).Export("hostValueAsInt64").
		NewFunctionBuilder().WithFunc(f.hostValueAsFloat32).Export("hostValueAsFloat32").
		NewFunctionBuilder().WithFunc(f.hostValueAsFloat64).Export("hostValueAsFloat64").
		NewFunctionBuilder().WithFunc(f.hostValueAsQNamePkg).Export("hostValueAsQNamePkg").
		NewFunctionBuilder().WithFunc(f.hostValueAsQNameEntity).Export("hostValueAsQNameEntity").
		NewFunctionBuilder().WithFunc(f.hostValueAsBool).Export("hostValueAsBool").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsBytes).Export("hostValueGetAsBytes").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsString).Export("hostValueGetAsString").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsInt32).Export("hostValueGetAsInt32").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsInt64).Export("hostValueGetAsInt64").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsFloat32).Export("hostValueGetAsFloat32").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsFloat64).Export("hostValueGetAsFloat64").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsValue).Export("hostValueGetAsValue").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsQNamePkg).Export("hostValueGetAsQNamePkg").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsQNameEntity).Export("hostValueGetAsQNameEntity").
		NewFunctionBuilder().WithFunc(f.hostValueGetAsBool).Export("hostValueGetAsBool").
		// Intents
		NewFunctionBuilder().WithFunc(f.hostNewValue).Export("hostNewValue").
		NewFunctionBuilder().WithFunc(f.hostUpdateValue).Export("hostUpdateValue").
		// RowWriters
		NewFunctionBuilder().WithFunc(f.hostRowWriterPutString).Export("hostRowWriterPutString").
		NewFunctionBuilder().WithFunc(f.hostRowWriterPutBytes).Export("hostRowWriterPutBytes").
		NewFunctionBuilder().WithFunc(f.hostRowWriterPutInt32).Export("hostRowWriterPutInt32").
		NewFunctionBuilder().WithFunc(f.hostRowWriterPutInt64).Export("hostRowWriterPutInt64").
		NewFunctionBuilder().WithFunc(f.hostRowWriterPutFloat32).Export("hostRowWriterPutFloat32").
		NewFunctionBuilder().WithFunc(f.hostRowWriterPutFloat64).Export("hostRowWriterPutFloat64").
		NewFunctionBuilder().WithFunc(f.hostRowWriterPutBool).Export("hostRowWriterPutBool").
		NewFunctionBuilder().WithFunc(f.hostRowWriterPutQName).Export("hostRowWriterPutQName").
		//ExportFunction("printstr", f.printStr).

		Instantiate(ctx)
	if err != nil {
		return err
	}

	return nil

}

func (f *wazeroExtEngine) resetModule(ctx context.Context, ePkg *wazeroExtPkg) (err error) {
	if ePkg.module != nil {
		err = ePkg.module.Close(ctx)
		if err != nil {
			return err
		}
		ePkg.stdout.buf = ePkg.stdout.buf[:0]
	}
	if f.compile {
		ePkg.module, err = f.rtm.InstantiateModule(ctx, ePkg.compiled, ePkg.moduleCfg)
	} else {
		ePkg.module, err = f.rtm.InstantiateWithConfig(ctx, ePkg.wasmData, ePkg.moduleCfg)
	}

	if err != nil {
		return err
	}

	err = ePkg.importFuncs(map[string]*api.Function{
		"malloc":               &ePkg.funcMalloc,
		"free":                 &ePkg.funcFree,
		"WasmAbiVersion_0_0_1": &ePkg.funcVer,
		"WasmGetHeapInuse":     &ePkg.funcGetHeapInuse,
		"WasmGetHeapSys":       &ePkg.funcGetHeapSys,
		"WasmGetMallocs":       &ePkg.funcGetMallocs,
		"WasmGetFrees":         &ePkg.funcGetFrees,
		"WasmGC":               &ePkg.funcGc,
		"WasmOnReadValue":      &ePkg.funcOnReadValue,
	})
	if err != nil {
		return err
	}

	res, err := ePkg.funcMalloc.Call(ctx, uint64(WasmPreallocatedBufferSize))
	if err != nil {
		return err
	}
	ePkg.allocatedBufs = append(ePkg.allocatedBufs, &allocatedBuf{
		addr: uint32(res[0]),
		offs: 0,
		cap:  WasmPreallocatedBufferSize,
	})

	for name := range ePkg.exts {
		expFunc := ePkg.module.ExportedFunction(name)
		if expFunc != nil {
			ePkg.exts[name] = expFunc
		} else {
			return missingExportedFunction(name)
		}
	}

	return nil
}

func (f *wazeroExtEngine) initModule(ctx context.Context, pkgName string, wasmdata []byte, extNames []string) (err error) {
	ePkg := &wazeroExtPkg{}

	ePkg.stdout = newLimitedWriter(maxStdErrSize)
	ePkg.moduleCfg = wazero.NewModuleConfig().WithName("wasm").WithStdout(&ePkg.stdout).WithSysWalltime().WithRandSource(rand.Reader)

	if f.compile {
		ePkg.compiled, err = f.rtm.CompileModule(ctx, wasmdata)
		if err != nil {
			return err
		}
	} else {
		ePkg.wasmData = wasmdata
	}

	ePkg.exts = make(map[string]api.Function)

	for _, name := range extNames {
		if !strings.HasPrefix(name, "Wasm") && name != "alloc" && name != "free" &&
			name != "calloc" && name != "realloc" && name != "malloc" && name != "_start" && name != "memory" {
			ePkg.exts[name] = nil // put to map to init later
		} else {
			return incorrectExtensionName(name)
		}
	}

	if err = f.resetModule(ctx, ePkg); err != nil {
		return err
	}

	// Check WASM SDK version
	_, err = ePkg.funcVer.Call(ctx)
	if err != nil {
		return errors.New("unsupported WASM version")
	}

	ePkg.pkgName = pkgName
	f.modules[pkgName] = ePkg

	return nil
}

func (f *wazeroExtEngine) Close(ctx context.Context) {
	for _, m := range f.modules {
		if m.module != nil {
			m.module.Close(ctx)
		}
	}
	if f.host != nil {
		f.host.Close(ctx)
	}
	if f.wasiCloser != nil {
		f.wasiCloser.Close(ctx)
	}
}

func (f *wazeroExtEngine) recover(ctx context.Context) {
	if err := f.resetModule(ctx, f.pkg); err != nil {
		panic(err)
	}
	if logger.IsInfo() {
		logger.Info(fmt.Sprintf("%s/%s wazero engine recovered", f.app.String(), f.pkg.pkgName))
	}
}

func (f *wazeroExtEngine) selectModule(pkgPath string) error {
	pkg, ok := f.modules[pkgPath]
	if !ok {
		return errUndefinedPackage(pkgPath)
	}
	f.pkg = pkg
	return nil
}

func (f *wazeroExtEngine) isMemoryOverflow(err error) bool {
	return strings.Contains(err.Error(), "runtime.alloc")
}

func (f *wazeroExtEngine) isPanic(err error) bool {
	return strings.Contains(err.Error(), "wasm error: unreachable")
}

func (f *wazeroExtEngine) invoke(ctx context.Context, extension appdef.FullQName, io iextengine.IExtensionIO) (err error) {
	var ok bool
	f.pkg, ok = f.modules[extension.PkgPath()]
	if !ok {
		return errUndefinedPackage(extension.PkgPath())
	}

	funct := f.pkg.exts[extension.Entity()]
	if funct == nil {
		return invalidExtensionName(extension.Entity())
	}

	f.safeApi = safestate.Provide(io, f.safeApi)
	f.ctx = ctx

	for i := range f.pkg.allocatedBufs {
		f.pkg.allocatedBufs[i].offs = 0 // reuse pre-allocated memory
	}

	f.pkg.stdout.Reset()

	begin := time.Now()

	_, err = funct.Call(ctx)

	if f.invocationsSeconds != nil {
		f.invocationsSeconds.Increase(time.Since(begin).Seconds())
	}

	if f.invocationsTotal != nil {
		f.invocationsTotal.Increase(1.0)
	}

	if err != nil && f.errorsTotal != nil {
		f.errorsTotal.Increase(1.0)
	}

	return err
}

func (f *wazeroExtEngine) Invoke(ctx context.Context, extension appdef.FullQName, io iextengine.IExtensionIO) (err error) {
	err = f.invoke(ctx, extension, io)
	if err != nil && f.isMemoryOverflow(err) && f.autoRecover {
		f.recover(ctx)
		if f.recoversTotal != nil {
			f.recoversTotal.Increase(1.0)
		}
		err = f.invoke(ctx, extension, io)
	}
	if err != nil && f.isPanic(err) && len(f.pkg.stdout.buf) > 0 {
		stdout := string(f.pkg.stdout.buf)
		if strings.HasPrefix(stdout, "panic: ") {
			return errors.New(strings.TrimSpace(stdout))
		}
	}
	return err
}

func (f *wazeroExtEngine) decodeStr(ptr, size uint32) string {
	if bytes, ok := f.pkg.module.Memory().Read(ptr, size); ok {
		return string(bytes)
	}
	panic(ErrUnableToReadMemory)
}

func (f *wazeroExtEngine) hostGetKey(storagePtr, storageSize, entityPtr, entitySize uint32) (res uint64) {
	storageFull := f.decodeStr(storagePtr, storageSize)
	entitystr := f.decodeStr(entityPtr, entitySize)
	return uint64(f.safeApi.KeyBuilder(storageFull, entitystr))
}

func (f *wazeroExtEngine) hostReadValues(keyID uint64) {
	f.safeApi.ReadValues(safe.TKeyBuilder(keyID), func(key safe.TKey, value safe.TValue) {
		_, err := f.pkg.funcOnReadValue.Call(f.ctx, uint64(key), uint64(value))
		if err != nil {
			panic(err.Error())
		}
	})
}

func (f *wazeroExtEngine) hostMustExist(keyID uint64) (result uint64) {
	return uint64(f.safeApi.MustGetValue(safe.TKeyBuilder(keyID)))
}

const maxUint64 = ^uint64(0)

func (f *wazeroExtEngine) hostCanExist(keyID uint64) (result uint64) {
	v, ok := f.safeApi.QueryValue(safe.TKeyBuilder(keyID))
	if !ok {
		return maxUint64
	}
	return uint64(v)
}

func (f *wazeroExtEngine) allocAndSend(buf []byte) (result uint64) {
	addrPkg, e := f.allocBuf(uint32(len(buf)))
	if e != nil {
		panic(e)
	}
	if !f.pkg.module.Memory().Write(addrPkg, buf) {
		panic(errMemoryOutOfRange)
	}
	return (uint64(addrPkg) << uint64(bitsInFourBytes)) | uint64(len(buf))
}

func (f *wazeroExtEngine) hostKeyAsString(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	v := f.safeApi.KeyAsString(safe.TKey(id), f.decodeStr(namePtr, nameSize))
	return f.allocAndSend([]byte(v))
}

func (f *wazeroExtEngine) hostKeyAsBytes(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	v := f.safeApi.KeyAsBytes(safe.TKey(id), f.decodeStr(namePtr, nameSize))
	return f.allocAndSend(v)
}

func (f *wazeroExtEngine) hostKeyAsInt32(id uint64, namePtr uint32, nameSize uint32) (result uint32) {
	return uint32(f.safeApi.KeyAsInt32(safe.TKey(id), f.decodeStr(namePtr, nameSize)))
}

func (f *wazeroExtEngine) hostKeyAsInt64(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	return uint64(f.safeApi.KeyAsInt64(safe.TKey(id), f.decodeStr(namePtr, nameSize)))
}

func (f *wazeroExtEngine) hostKeyAsBool(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	b := f.safeApi.KeyAsBool(safe.TKey(id), f.decodeStr(namePtr, nameSize))
	if b {
		return uint64(1)
	}
	return uint64(0)
}

func (f *wazeroExtEngine) hostKeyAsQNamePkg(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	qname := f.safeApi.KeyAsQName(safe.TKey(id), f.decodeStr(namePtr, nameSize))
	return f.allocAndSend([]byte(qname.FullPkgName))
}

func (f *wazeroExtEngine) hostKeyAsQNameEntity(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	qname := f.safeApi.KeyAsQName(safe.TKey(id), f.decodeStr(namePtr, nameSize))
	return f.allocAndSend([]byte(qname.Entity))
}

func (f *wazeroExtEngine) hostKeyAsFloat32(key uint64, namePtr uint32, nameSize uint32) (result float32) {
	return f.safeApi.KeyAsFloat32(safe.TKey(key), f.decodeStr(namePtr, nameSize))
}

func (f *wazeroExtEngine) hostKeyAsFloat64(key uint64, namePtr uint32, nameSize uint32) (result float64) {
	return f.safeApi.KeyAsFloat64(safe.TKey(key), f.decodeStr(namePtr, nameSize))
}

func (f *wazeroExtEngine) hostValueGetAsString(value uint64, index uint32) (result uint64) {
	v := f.safeApi.ValueGetAsString(safe.TValue(value), int(index))
	return f.allocAndSend([]byte(v))
}

func (f *wazeroExtEngine) hostValueGetAsQNameEntity(value uint64, index uint32) (result uint64) {
	qname := f.safeApi.ValueGetAsQName(safe.TValue(value), int(index))
	return f.allocAndSend([]byte(qname.Entity))
}

func (f *wazeroExtEngine) hostValueGetAsQNamePkg(value uint64, index uint32) (result uint64) {
	qname := f.safeApi.ValueGetAsQName(safe.TValue(value), int(index))
	return f.allocAndSend([]byte(qname.FullPkgName))
}

func (f *wazeroExtEngine) hostValueGetAsBytes(value uint64, index uint32) (result uint64) {
	return f.allocAndSend(f.safeApi.ValueGetAsBytes(safe.TValue(value), int(index)))
}

func (f *wazeroExtEngine) hostValueGetAsBool(value uint64, index uint32) (result uint64) {
	b := f.safeApi.ValueGetAsBool(safe.TValue(value), int(index))
	if b {
		return 1
	}
	return 0
}

func (f *wazeroExtEngine) hostValueGetAsInt32(value uint64, index uint32) (result int32) {
	return f.safeApi.ValueGetAsInt32(safe.TValue(value), int(index))
}

func (f *wazeroExtEngine) hostValueGetAsInt64(value uint64, index uint32) (result uint64) {
	return uint64(f.safeApi.ValueGetAsInt64(safe.TValue(value), int(index)))
}

func (f *wazeroExtEngine) hostValueGetAsFloat32(id uint64, index uint32) float32 {
	return f.safeApi.ValueGetAsFloat32(safe.TValue(id), int(index))
}

func (f *wazeroExtEngine) hostValueGetAsFloat64(id uint64, index uint32) float64 {
	return f.safeApi.ValueGetAsFloat64(safe.TValue(id), int(index))
}

func (f *wazeroExtEngine) hostValueGetAsValue(val uint64, index uint32) (result uint64) {
	return uint64(f.safeApi.ValueGetAsValue(safe.TValue(val), int(index)))
}

func (f *wazeroExtEngine) hostValueAsString(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	s := f.safeApi.ValueAsString(safe.TValue(id), f.decodeStr(namePtr, nameSize))
	return f.allocAndSend([]byte(s))
}

func (f *wazeroExtEngine) hostValueAsBytes(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	b := f.safeApi.ValueAsBytes(safe.TValue(id), f.decodeStr(namePtr, nameSize))
	return f.allocAndSend(b)
}

func (f *wazeroExtEngine) hostValueAsInt32(id uint64, namePtr uint32, nameSize uint32) (result int32) {
	return f.safeApi.ValueAsInt32(safe.TValue(id), f.decodeStr(namePtr, nameSize))
}

func (f *wazeroExtEngine) hostValueAsInt64(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	return uint64(f.safeApi.ValueAsInt64(safe.TValue(id), f.decodeStr(namePtr, nameSize)))
}

func (f *wazeroExtEngine) hostValueAsBool(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	b := f.safeApi.ValueAsBool(safe.TValue(id), f.decodeStr(namePtr, nameSize))
	if b {
		return 1
	}
	return 0
}

func (f *wazeroExtEngine) hostValueAsFloat32(id uint64, namePtr, nameSize uint32) float32 {
	return f.safeApi.ValueAsFloat32(safe.TValue(id), f.decodeStr(namePtr, nameSize))
}

func (f *wazeroExtEngine) hostValueAsFloat64(id uint64, namePtr, nameSize uint32) float64 {
	return f.safeApi.ValueAsFloat64(safe.TValue(id), f.decodeStr(namePtr, nameSize))
}

func (f *wazeroExtEngine) hostValueAsQNameEntity(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	qname := f.safeApi.ValueAsQName(safe.TValue(id), f.decodeStr(namePtr, nameSize))
	return f.allocAndSend([]byte(qname.Entity))
}

func (f *wazeroExtEngine) hostValueAsQNamePkg(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	qname := f.safeApi.ValueAsQName(safe.TValue(id), f.decodeStr(namePtr, nameSize))
	return f.allocAndSend([]byte(qname.FullPkgName))
}

func (f *wazeroExtEngine) hostValueAsValue(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	return uint64(f.safeApi.ValueAsValue(safe.TValue(id), f.decodeStr(namePtr, nameSize)))
}

func (f *wazeroExtEngine) hostValueLength(id uint64) (result uint32) {
	return uint32(f.safeApi.ValueLen(safe.TValue(id)))
}

func (f *wazeroExtEngine) allocBuf(size uint32) (addr uint32, err error) {
	for i := range f.pkg.allocatedBufs {
		if f.pkg.allocatedBufs[i].cap-f.pkg.allocatedBufs[i].offs >= size {
			addr = f.pkg.allocatedBufs[i].addr + f.pkg.allocatedBufs[i].offs
			f.pkg.allocatedBufs[i].offs += size
			return
		}
	}
	// no space in the allocated buffers

	var newBufferSize uint32 = WasmPreallocatedBufferIncrease
	if size > newBufferSize {
		newBufferSize = size
	}

	var res []uint64
	res, err = f.pkg.funcMalloc.Call(f.ctx, uint64(newBufferSize))
	if err != nil {
		return 0, err
	}
	addr = uint32(res[0]) // nolint G115
	f.pkg.allocatedBufs = append(f.pkg.allocatedBufs, &allocatedBuf{
		addr: addr,
		offs: 0,
		cap:  newBufferSize,
	})
	return addr, nil
}

func (f *wazeroExtEngine) getFrees(packagePath string, ctx context.Context) (uint64, error) {
	pkg, ok := f.modules[packagePath]
	if !ok {
		return 0, errUndefinedPackage(packagePath)
	}
	res, err := pkg.funcGetFrees.Call(ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) gc(packagePath string, ctx context.Context) error {
	pkg, ok := f.modules[packagePath]
	if !ok {
		return errUndefinedPackage(packagePath)
	}
	_, err := pkg.funcGc.Call(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (f *wazeroExtEngine) getHeapinuse(packagePath string, ctx context.Context) (uint64, error) {
	pkg, ok := f.modules[packagePath]
	if !ok {
		return 0, errUndefinedPackage(packagePath)
	}
	res, err := pkg.funcGetHeapInuse.Call(ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) getHeapSys(packagePath string, ctx context.Context) (uint64, error) {
	pkg, ok := f.modules[packagePath]
	if !ok {
		return 0, errUndefinedPackage(packagePath)
	}
	res, err := pkg.funcGetHeapSys.Call(ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) getMallocs(packagePath string, ctx context.Context) (uint64, error) {
	pkg, ok := f.modules[packagePath]
	if !ok {
		return 0, errUndefinedPackage(packagePath)
	}
	res, err := pkg.funcGetMallocs.Call(ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) hostNewValue(keyID uint64) uint64 {
	return uint64(f.safeApi.NewValue(safe.TKeyBuilder(keyID)))
}

func (f *wazeroExtEngine) hostUpdateValue(keyID, existingValueID uint64) (result uint64) {
	return uint64(f.safeApi.UpdateValue(safe.TKeyBuilder(keyID), safe.TValue(existingValueID)))
}

func (f *wazeroExtEngine) hostRowWriterPutString(id uint64, typ uint32, namePtr uint32, nameSize, valuePtr, valueSize uint32) {
	if typ == 0 {
		f.safeApi.KeyBuilderPutString(safe.TKeyBuilder(id), f.decodeStr(namePtr, nameSize), f.decodeStr(valuePtr, valueSize))
	} else {
		f.safeApi.IntentPutString(safe.TIntent(id), f.decodeStr(namePtr, nameSize), f.decodeStr(valuePtr, valueSize))
	}
}

func (f *wazeroExtEngine) hostRowWriterPutBytes(id uint64, typ uint32, namePtr uint32, nameSize, valuePtr, valueSize uint32) {
	var bytes []byte
	var ok bool
	bytes, ok = f.pkg.module.Memory().Read(valuePtr, valueSize)
	if !ok {
		panic(ErrUnableToReadMemory)
	}
	if typ == 0 {
		f.safeApi.KeyBuilderPutBytes(safe.TKeyBuilder(id), f.decodeStr(namePtr, nameSize), bytes)
	} else {
		f.safeApi.IntentPutBytes(safe.TIntent(id), f.decodeStr(namePtr, nameSize), bytes)
	}
}

func (f *wazeroExtEngine) hostRowWriterPutInt32(id uint64, typ uint32, namePtr uint32, nameSize uint32, value int32) {
	if typ == 0 {
		f.safeApi.KeyBuilderPutInt32(safe.TKeyBuilder(id), f.decodeStr(namePtr, nameSize), value)
	} else {
		f.safeApi.IntentPutInt32(safe.TIntent(id), f.decodeStr(namePtr, nameSize), value)
	}
}

func (f *wazeroExtEngine) hostRowWriterPutInt64(id uint64, typ uint32, namePtr uint32, nameSize uint32, value int64) {
	if typ == 0 {
		f.safeApi.KeyBuilderPutInt64(safe.TKeyBuilder(id), f.decodeStr(namePtr, nameSize), value)
	} else {
		f.safeApi.IntentPutInt64(safe.TIntent(id), f.decodeStr(namePtr, nameSize), value)
	}
}

func (f *wazeroExtEngine) hostRowWriterPutQName(id uint64, typ uint32, namePtr uint32, nameSize uint32, pkgPtr, pkgSize, entityPtr, entitySize uint32) {
	qname := safe.QName{
		FullPkgName: f.decodeStr(pkgPtr, pkgSize),
		Entity:      f.decodeStr(entityPtr, entitySize),
	}
	if typ == 0 {
		f.safeApi.KeyBuilderPutQName(safe.TKeyBuilder(id), f.decodeStr(namePtr, nameSize), qname)
	} else {
		f.safeApi.IntentPutQName(safe.TIntent(id), f.decodeStr(namePtr, nameSize), qname)
	}
}

func (f *wazeroExtEngine) hostRowWriterPutBool(id uint64, typ uint32, namePtr uint32, nameSize uint32, value int32) {
	if typ == 0 {
		f.safeApi.KeyBuilderPutBool(safe.TKeyBuilder(id), f.decodeStr(namePtr, nameSize), value > 0)
	} else {
		f.safeApi.IntentPutBool(safe.TIntent(id), f.decodeStr(namePtr, nameSize), value > 0)
	}
}

func (f *wazeroExtEngine) hostRowWriterPutFloat32(id uint64, typ uint32, namePtr uint32, nameSize uint32, value float32) {
	if typ == 0 {
		f.safeApi.KeyBuilderPutFloat32(safe.TKeyBuilder(id), f.decodeStr(namePtr, nameSize), value)
	} else {
		f.safeApi.IntentPutFloat32(safe.TIntent(id), f.decodeStr(namePtr, nameSize), value)
	}
}

func (f *wazeroExtEngine) hostRowWriterPutFloat64(id uint64, typ uint32, namePtr, nameSize uint32, value float64) {
	if typ == 0 {
		f.safeApi.KeyBuilderPutFloat64(safe.TKeyBuilder(id), f.decodeStr(namePtr, nameSize), value)
	} else {
		f.safeApi.IntentPutFloat64(safe.TIntent(id), f.decodeStr(namePtr, nameSize), value)
	}
}
