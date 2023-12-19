/*
  - Copyright (c) 2023-present unTill Software Development Group B. V.
    @author Michael Saigachenko
*/

package iextenginewasm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	"github.com/voedger/voedger/pkg/istructs"
)

type wazeroExtPkg struct {
	module api.Module
	exts   map[string]api.Function

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
	recoverMem    []byte
}

type wazeroExtEngine struct {
	compile bool
	config  *iextengine.ExtEngineConfig
	modules map[string]*wazeroExtPkg
	host    api.Module
	rtm     wazero.Runtime
	//recoverMem api.Memory

	//mwasi api.Module
	// ce    api.ICallEngine
	// cep   api.CallEngineParams
	keys          []istructs.IKey
	keyBuilders   []istructs.IStateKeyBuilder
	values        []istructs.IStateValue
	valueBuilders []istructs.IStateValueBuilder
	wasiCloser    api.Closer

	// Invoke-related!
	io  iextengine.IExtensionIO
	ctx context.Context
	pkg *wazeroExtPkg
}

type allocatedBuf struct {
	addr uint32
	offs uint32
	cap  uint32
}

type extensionEngineFactory struct {
	compile bool
}

func (f extensionEngineFactory) New(ctx context.Context, packages []iextengine.ExtensionPackage, config *iextengine.ExtEngineConfig, numEngines int) (engines []iextengine.IExtensionEngine, err error) {
	for i := 0; i < numEngines; i++ {
		engine := &wazeroExtEngine{
			modules: make(map[string]*wazeroExtPkg),
			config:  config,
			compile: f.compile,
		}
		err = engine.init(ctx)
		if err != nil {
			return engines, err
		}
		engines = append(engines, engine)
	}

	for _, pkg := range packages {
		if pkg.ModuleUrl.Scheme == "file" && (pkg.ModuleUrl.Host == "" || strings.EqualFold("localhost", pkg.ModuleUrl.Scheme)) {
			path := pkg.ModuleUrl.Path
			if runtime.GOOS == "windows" {
				path = strings.TrimPrefix(path, "/")
			}

			wasmdata, err := os.ReadFile(path)

			if err != nil {
				return nil, err
			}

			for _, eng := range engines {
				err = eng.(*wazeroExtEngine).initModule(ctx, pkg.QualifiedName, wasmdata, pkg.ExtensionNames)
				if err != nil {
					return nil, err
				}
			}
		} else {
			return nil, fmt.Errorf("unsupported URL: " + pkg.ModuleUrl.String())
		}
	}
	return engines, nil
}

func (f *wazeroExtEngine) SetLimits(limits iextengine.ExtensionLimits) {
	//f.cep.Duration = limits.ExecutionInterval
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
	if uint32(memoryLimit) <= uint32(limit) {
		return fmt.Errorf("the minimum limit of memory is: %.1f bytes, requested limit is: %.1f", limit, float32(memoryLimit))
	}

	var rtConf wazero.RuntimeConfig

	if f.compile {
		rtConf = wazero.NewRuntimeConfigCompiler()
	} else {
		rtConf = wazero.NewRuntimeConfigInterpreter()
	}
	rtConf = rtConf.
		WithCoreFeatures(api.CoreFeatureBulkMemoryOperations).
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
		NewFunctionBuilder().WithFunc(f.hostPanic).Export("hostPanic").
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

	f.keyBuilders = make([]istructs.IStateKeyBuilder, 0, keysBuildersCapacity)
	f.values = make([]istructs.IStateValue, 0, valuesCapacity)
	f.valueBuilders = make([]istructs.IStateValueBuilder, 0, valueBuildersCapacity)

	return nil

}

func (f *wazeroExtEngine) initModule(ctx context.Context, pkgName string, wasmdata []byte, extNames []string) (err error) {
	ePkg := &wazeroExtPkg{}

	moduleCfg := wazero.NewModuleConfig().WithName("wasm").WithStdout(io.Discard).WithStderr(io.Discard)
	if f.compile {
		compiledWasm, err := f.rtm.CompileModule(ctx, wasmdata)
		if err != nil {
			return err
		}

		ePkg.module, err = f.rtm.InstantiateModule(ctx, compiledWasm, moduleCfg)
		if err != nil {
			return err
		}
	} else {
		ePkg.module, err = f.rtm.InstantiateWithConfig(ctx, wasmdata, moduleCfg)
		if err != nil {
			return err
		}
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

	// Check WASM SDK version
	_, err = ePkg.funcVer.Call(ctx)
	if err != nil {
		return errors.New("unsupported WASM version")
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

	backup, read := ePkg.module.Memory().Read(0, ePkg.module.Memory().Size())
	if !read {
		return fmt.Errorf("unable to backup memory")
	}

	ePkg.recoverMem = make([]byte, ePkg.module.Memory().Size())
	copy(ePkg.recoverMem[0:], backup[0:])

	ePkg.exts = make(map[string]api.Function)

	for _, name := range extNames {
		if !strings.HasPrefix(name, "Wasm") && name != "alloc" && name != "free" &&
			name != "calloc" && name != "realloc" && name != "malloc" && name != "_start" && name != "memory" {
			expFunc := ePkg.module.ExportedFunction(name)
			if expFunc != nil {
				ePkg.exts[name] = expFunc
			} else {
				return missingExportedFunction(name)
			}
		} else {
			return incorrectExtensionName(name)
		}
	}

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

func (f *wazeroExtEngine) recover() {
	if !f.pkg.module.Memory().Write(0, f.pkg.recoverMem) {
		panic("unable to restore memory")
	}
}

func (f *wazeroExtEngine) Invoke(ctx context.Context, extension iextengine.ExtQName, io iextengine.IExtensionIO) (err error) {

	var ok bool
	f.pkg, ok = f.modules[extension.PackageName]
	if !ok {
		return undefinedPackage(extension.PackageName)
	}

	funct := f.pkg.exts[extension.ExtName]
	if funct == nil {
		return invalidExtensionName(extension.ExtName)
	}

	f.io = io
	f.ctx = ctx

	if len(f.keys) > 0 {
		f.keys = make([]istructs.IKey, 0, keysCapacity)
	}
	if len(f.keyBuilders) > 0 {
		f.keyBuilders = make([]istructs.IStateKeyBuilder, 0, keysBuildersCapacity)
	}
	if len(f.values) > 0 {
		f.values = make([]istructs.IStateValue, 0, valuesCapacity)
	}
	if len(f.valueBuilders) > 0 {
		f.valueBuilders = make([]istructs.IStateValueBuilder, 0, valueBuildersCapacity)
	}
	for i := range f.pkg.allocatedBufs {
		f.pkg.allocatedBufs[i].offs = 0 // reuse pre-allocated memory
	}

	_, err = funct.Call(ctx)

	if err != nil {
		f.recover()
	}

	return err
}

func (f *wazeroExtEngine) decodeStr(ptr, size uint32) string {
	if bytes, ok := f.pkg.module.Memory().Read(uint32(ptr), uint32(size)); ok {
		return string(bytes)
	}
	panic(ErrUnableToReadMemory)
}

func (f *wazeroExtEngine) hostGetKey(storagePtr, storageSize, entityPtr, entitySize uint32) (res uint64) {

	var storage appdef.QName
	var entity appdef.QName
	var err error
	storage, err = appdef.ParseQName(f.decodeStr(storagePtr, storageSize))
	if err != nil {
		panic(err)
	}
	entitystr := f.decodeStr(entityPtr, entitySize)
	if entitystr != "" {
		entity, err = appdef.ParseQName(entitystr)
		if err != nil {
			panic(err)
		}
	}
	k, e := f.io.KeyBuilder(storage, entity)
	if e != nil {
		panic(e)
	}
	res = uint64(len(f.keyBuilders))
	f.keyBuilders = append(f.keyBuilders, k)
	return
}

func (f *wazeroExtEngine) hostPanic(namePtr, nameSize uint32) {
	panic(f.decodeStr(namePtr, nameSize))
}

func (f *wazeroExtEngine) hostReadValues(keyId uint64) {
	if int(keyId) >= len(f.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	first := true
	keyIndex := len(f.keys)
	valueIndex := len(f.values)
	err := f.io.Read(f.keyBuilders[keyId], func(key istructs.IKey, value istructs.IStateValue) (err error) {
		if first {
			f.keys = append(f.keys, key)
			f.values = append(f.values, value)
			first = false
		} else { // replace
			f.keys[keyIndex] = key
			f.values[valueIndex] = value
		}
		_, err = f.pkg.funcOnReadValue.Call(f.ctx, uint64(keyIndex), uint64(valueIndex))
		return err
	})
	if err != nil {
		panic(err.Error())
	}
}

func (f *wazeroExtEngine) hostMustExist(keyId uint64) (result uint64) {

	if int(keyId) >= len(f.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	v, e := f.io.MustExist(f.keyBuilders[keyId])
	if e != nil {
		panic(e)
	}
	result = uint64(len(f.values))
	f.values = append(f.values, v)
	return
}

const maxUint64 = ^uint64(0)

func (f *wazeroExtEngine) hostCanExist(keyId uint64) (result uint64) {
	if int(keyId) >= len(f.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	v, ok, e := f.io.CanExist(f.keyBuilders[keyId])
	if e != nil {
		panic(e)
	}
	if !ok {
		return maxUint64
	}
	result = uint64(len(f.values))
	f.values = append(f.values, v)
	return
}

func (f *wazeroExtEngine) allocAndSend(buf []byte) (result uint64) {
	addrPkg, e := f.allocBuf(uint32(len(buf)))
	if e != nil {
		panic(e)
	}
	if !f.pkg.module.Memory().Write(addrPkg, buf) {
		panic(e)
	}
	return (uint64(addrPkg) << uint64(bitsInFourBytes)) | uint64(len(buf))
}

func (f *wazeroExtEngine) keyargs(id uint64, namePtr uint32, nameSize uint32) (istructs.IKey, string) {
	if int(id) >= len(f.keys) {
		panic(PanicIncorrectKey)
	}
	return f.keys[id], f.decodeStr(namePtr, nameSize)
}

func (f *wazeroExtEngine) valueargs(id uint64, namePtr uint32, nameSize uint32) (istructs.IStateValue, string) {
	if int(id) >= len(f.values) {
		panic(PanicIncorrectValue)
	}
	return f.values[id], f.decodeStr(namePtr, nameSize)
}

func (f *wazeroExtEngine) value(id uint64) istructs.IStateValue {
	if int(id) >= len(f.values) {
		panic(PanicIncorrectValue)
	}
	return f.values[id]
}

func (f *wazeroExtEngine) hostKeyAsString(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	key, name := f.keyargs(id, namePtr, nameSize)
	return f.allocAndSend([]byte(key.AsString(name)))
}

func (f *wazeroExtEngine) hostKeyAsBytes(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	key, name := f.keyargs(id, namePtr, nameSize)
	return f.allocAndSend(key.AsBytes(name))
}

func (f *wazeroExtEngine) hostKeyAsInt32(id uint64, namePtr uint32, nameSize uint32) (result uint32) {
	key, name := f.keyargs(id, namePtr, nameSize)
	return uint32(key.AsInt32(name))
}

func (f *wazeroExtEngine) hostKeyAsInt64(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	key, name := f.keyargs(id, namePtr, nameSize)
	return uint64(key.AsInt64(name))
}

func (f *wazeroExtEngine) hostKeyAsBool(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	key, name := f.keyargs(id, namePtr, nameSize)
	if key.AsBool(name) {
		return uint64(1)
	}
	return uint64(0)
}

func (f *wazeroExtEngine) hostKeyAsQNamePkg(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	key, name := f.keyargs(id, namePtr, nameSize)
	qname := key.AsQName(name)
	return f.allocAndSend([]byte(qname.Pkg()))
}

func (f *wazeroExtEngine) hostKeyAsQNameEntity(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	key, name := f.keyargs(id, namePtr, nameSize)
	qname := key.AsQName(name)
	return f.allocAndSend([]byte(qname.Entity()))
}

func (f *wazeroExtEngine) hostKeyAsFloat32(key uint64, namePtr uint32, nameSize uint32) (result float32) {
	k, name := f.keyargs(key, namePtr, nameSize)
	return k.AsFloat32(name)
}

func (f *wazeroExtEngine) hostKeyAsFloat64(key uint64, namePtr uint32, nameSize uint32) (result float64) {
	k, name := f.keyargs(key, namePtr, nameSize)
	return k.AsFloat64(name)
}

func (f *wazeroExtEngine) hostValueGetAsString(value uint64, index uint32) (result uint64) {
	v := f.value(value)
	return f.allocAndSend([]byte(v.GetAsString(int(index))))
}

func (f *wazeroExtEngine) hostValueGetAsQNameEntity(value uint64, index uint32) (result uint64) {
	v := f.value(value)
	qname := v.GetAsQName(int(index))
	return f.allocAndSend([]byte(qname.Entity()))
}

func (f *wazeroExtEngine) hostValueGetAsQNamePkg(value uint64, index uint32) (result uint64) {
	v := f.value(value)
	qname := v.GetAsQName(int(index))
	return f.allocAndSend([]byte(qname.Pkg()))
}

func (f *wazeroExtEngine) hostValueGetAsBytes(value uint64, index uint32) (result uint64) {
	v := f.value(value)
	return f.allocAndSend(v.GetAsBytes(int(index)))
}

func (f *wazeroExtEngine) hostValueGetAsBool(value uint64, index uint32) (result uint64) {
	v := f.value(value)
	if v.GetAsBool(int(index)) {
		return 1
	}
	return 0
}

func (f *wazeroExtEngine) hostValueGetAsInt32(value uint64, index uint32) (result int32) {
	v := f.value(value)
	return v.GetAsInt32(int(index))
}

func (f *wazeroExtEngine) hostValueGetAsInt64(value uint64, index uint32) (result uint64) {
	v := f.value(value)
	return uint64(v.GetAsInt64(int(index)))
}

func (f *wazeroExtEngine) hostValueGetAsFloat32(id uint64, index uint32) float32 {
	return f.value(id).GetAsFloat32(int(index))
}

func (f *wazeroExtEngine) hostValueGetAsFloat64(id uint64, index uint32) float64 {
	return f.value(id).GetAsFloat64(int(index))
}

func (f *wazeroExtEngine) hostValueGetAsValue(val uint64, index uint32) (result uint64) {
	v := f.value(val)
	value := v.GetAsValue(int(index))
	result = uint64(len(f.values))
	f.values = append(f.values, value)
	return
}

func (f *wazeroExtEngine) hostValueAsString(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	v, name := f.valueargs(id, namePtr, nameSize)
	return f.allocAndSend([]byte(v.AsString(name)))
}

func (f *wazeroExtEngine) hostValueAsBytes(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	v, name := f.valueargs(id, namePtr, nameSize)
	return f.allocAndSend(v.AsBytes(name))
}

func (f *wazeroExtEngine) hostValueAsInt32(id uint64, namePtr uint32, nameSize uint32) (result int32) {
	v, name := f.valueargs(id, namePtr, nameSize)
	return v.AsInt32(name)
}

func (f *wazeroExtEngine) hostValueAsInt64(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	v, name := f.valueargs(id, namePtr, nameSize)
	return uint64(v.AsInt64(name))
}

func (f *wazeroExtEngine) hostValueAsBool(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	v, name := f.valueargs(id, namePtr, nameSize)
	if v.AsBool(name) {
		return 1
	}
	return 0
}

func (f *wazeroExtEngine) hostValueAsFloat32(id uint64, namePtr, nameSize uint32) float32 {
	v, name := f.valueargs(id, namePtr, nameSize)
	return v.AsFloat32(name)
}

func (f *wazeroExtEngine) hostValueAsFloat64(id uint64, namePtr, nameSize uint32) float64 {
	v, name := f.valueargs(id, namePtr, nameSize)
	return v.AsFloat64(name)
}

func (f *wazeroExtEngine) hostValueAsQNameEntity(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	v, name := f.valueargs(id, namePtr, nameSize)
	qname := v.AsQName(name)
	return f.allocAndSend([]byte(qname.Entity()))
}

func (f *wazeroExtEngine) hostValueAsQNamePkg(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	v, name := f.valueargs(id, namePtr, nameSize)
	qname := v.AsQName(name)
	return f.allocAndSend([]byte(qname.Pkg()))
}

func (f *wazeroExtEngine) hostValueAsValue(id uint64, namePtr uint32, nameSize uint32) (result uint64) {
	v, name := f.valueargs(id, namePtr, nameSize)
	value := v.AsValue(name)
	result = uint64(len(f.values))
	f.values = append(f.values, value)
	return
}

func (f *wazeroExtEngine) hostValueLength(id uint64) (result uint32) {
	if int(id) >= len(f.values) {
		panic(PanicIncorrectValue)
	}
	return uint32(f.values[id].Length())
}

func (f *wazeroExtEngine) allocBuf(size uint32) (addr uint32, err error) {
	for i := range f.pkg.allocatedBufs {
		if f.pkg.allocatedBufs[i].cap-f.pkg.allocatedBufs[i].offs >= size {
			addr = f.pkg.allocatedBufs[i].addr + f.pkg.allocatedBufs[i].offs
			f.pkg.allocatedBufs[i].offs += uint32(size)
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
	addr = uint32(res[0])
	f.pkg.allocatedBufs = append(f.pkg.allocatedBufs, &allocatedBuf{
		addr: addr,
		offs: 0,
		cap:  newBufferSize,
	})
	return addr, nil
}

func (f *wazeroExtEngine) getFrees(packageName string, ctx context.Context) (uint64, error) {
	pkg, ok := f.modules[packageName]
	if !ok {
		return 0, undefinedPackage(packageName)
	}
	res, err := pkg.funcGetFrees.Call(ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) gc(packageName string, ctx context.Context) error {
	pkg, ok := f.modules[packageName]
	if !ok {
		return undefinedPackage(packageName)
	}
	_, err := pkg.funcGc.Call(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (f *wazeroExtEngine) getHeapinuse(packageName string, ctx context.Context) (uint64, error) {
	pkg, ok := f.modules[packageName]
	if !ok {
		return 0, undefinedPackage(packageName)
	}
	res, err := pkg.funcGetHeapInuse.Call(ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) getHeapSys(packageName string, ctx context.Context) (uint64, error) {
	pkg, ok := f.modules[packageName]
	if !ok {
		return 0, undefinedPackage(packageName)
	}
	res, err := pkg.funcGetHeapSys.Call(ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) getMallocs(packageName string, ctx context.Context) (uint64, error) {
	pkg, ok := f.modules[packageName]
	if !ok {
		return 0, undefinedPackage(packageName)
	}
	res, err := pkg.funcGetMallocs.Call(ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) hostNewValue(keyId uint64) (result uint64) {
	if int(keyId) >= len(f.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	vb, err := f.io.NewValue(f.keyBuilders[keyId])
	if err != nil {
		panic(err)
	}
	result = uint64(len(f.valueBuilders))
	f.valueBuilders = append(f.valueBuilders, vb)
	return
}

func (f *wazeroExtEngine) hostUpdateValue(keyId, existingValueId uint64) (result uint64) {
	if int(keyId) >= len(f.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	if int(existingValueId) >= len(f.values) {
		panic(PanicIncorrectValue)
	}
	vb, err := f.io.UpdateValue(f.keyBuilders[keyId], f.values[existingValueId])
	if err != nil {
		panic(err)
	}
	result = uint64(len(f.valueBuilders))
	f.valueBuilders = append(f.valueBuilders, vb)
	return
}

func (f *wazeroExtEngine) getWriterArgs(id uint64, typ uint32, namePtr uint32, nameSize uint32) (writer istructs.IRowWriter, name string) {
	switch typ {
	case 0:
		if int(id) >= len(f.keyBuilders) {
			panic(PanicIncorrectKeyBuilder)
		}
		writer = f.keyBuilders[id]
	default:
		if int(id) >= len(f.valueBuilders) {
			panic(PanicIncorrectIntent)
		}
		writer = f.valueBuilders[id]
	}
	name = f.decodeStr(namePtr, nameSize)
	return
}

func (f *wazeroExtEngine) hostRowWriterPutString(id uint64, typ uint32, namePtr uint32, nameSize, valuePtr, valueSize uint32) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)
	writer.PutString(name, f.decodeStr(valuePtr, valueSize))
}

func (f *wazeroExtEngine) hostRowWriterPutBytes(id uint64, typ uint32, namePtr uint32, nameSize, valuePtr, valueSize uint32) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)

	var bytes []byte
	var ok bool
	bytes, ok = f.pkg.module.Memory().Read(uint32(valuePtr), uint32(valueSize))
	if !ok {
		panic(ErrUnableToReadMemory)
	}

	writer.PutBytes(name, bytes)
}

func (f *wazeroExtEngine) hostRowWriterPutInt32(id uint64, typ uint32, namePtr uint32, nameSize uint32, value int32) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)
	writer.PutInt32(name, value)
}

func (f *wazeroExtEngine) hostRowWriterPutInt64(id uint64, typ uint32, namePtr uint32, nameSize uint32, value int64) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)
	writer.PutInt64(name, value)
}

func (f *wazeroExtEngine) hostRowWriterPutQName(id uint64, typ uint32, namePtr uint32, nameSize uint32, value int64, pkgPtr, pkgSize, entityPtr, entitySize uint32) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)
	pkg := f.decodeStr(pkgPtr, pkgSize)
	entity := f.decodeStr(entityPtr, entitySize)
	writer.PutQName(name, appdef.NewQName(pkg, entity))
}

func (f *wazeroExtEngine) hostRowWriterPutBool(id uint64, typ uint32, namePtr uint32, nameSize uint32, value int32) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)
	writer.PutBool(name, value > 0)
}

func (f *wazeroExtEngine) hostRowWriterPutFloat32(id uint64, typ uint32, namePtr uint32, nameSize uint32, value float32) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)
	writer.PutFloat32(name, value)
}

func (f *wazeroExtEngine) hostRowWriterPutFloat64(id uint64, typ uint32, namePtr, nameSize uint32, value float64) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)
	writer.PutFloat64(name, value)
}
