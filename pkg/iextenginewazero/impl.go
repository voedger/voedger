/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package iextenginewasm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/heeus/wazero"
	"github.com/heeus/wazero/api"
	"github.com/heeus/wazero/wasi"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iextengine"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

type wazeroExtEngine struct {
	data       []byte
	module     api.Module
	host       api.Module
	recoverMem api.Memory

	mwasi api.Module
	ce    api.ICallEngine
	cep   api.CallEngineParams

	allocatedBufs []*allocatedBuf

	keys          []istructs.IKey
	keyBuilders   []istructs.IStateKeyBuilder
	values        []istructs.IStateValue
	valueBuilders []istructs.IStateValueBuilder

	funcMalloc api.Function
	funcFree   api.Function

	funcVer          api.Function
	funcGetHeapInuse api.Function
	funcGetHeapSys   api.Function
	funcGetMallocs   api.Function
	funcGetFrees     api.Function
	funcGc           api.Function
	funcOnReadValue  api.Function

	ctx context.Context
	io  iextengine.IExtentionIO
}

type allocatedBuf struct {
	addr uint32
	offs uint32
	cap  uint32
}

type wasmExtension struct {
	name   string
	f      api.Function
	engine *wazeroExtEngine
}

func (ext *wasmExtension) Invoke(io iextengine.IExtentionIO) (err error) {
	return ext.engine.invoke(ext.f, io)
}

func ExtEngineWazeroFactory(ctx context.Context, moduleURL *url.URL, cfg iextengine.ExtEngineConfig) (e iextengine.IExtensionEngine, closer func(), err error) {

	var wasmdata []byte
	if moduleURL.Scheme == "file" && (moduleURL.Host == "" || strings.EqualFold("localhost", moduleURL.Scheme)) {
		path := moduleURL.Path
		if runtime.GOOS == "windows" {
			path = strings.TrimPrefix(path, "/")
		}

		wasmdata, err = os.ReadFile(path)
		if err != nil {
			return nil, func() {}, err
		}
	} else {
		return nil, func() {}, fmt.Errorf("unsupported URL: " + moduleURL.String())
	}

	impl := &wazeroExtEngine{data: wasmdata}

	err = impl.init(ctx, cfg)
	if err != nil {
		return nil, func() {}, err
	}

	return impl, impl.close, nil
}

func (f *wazeroExtEngine) SetLimits(limits iextengine.ExtensionLimits) {
	f.cep.Duration = limits.ExecutionInterval
}

func (f *wazeroExtEngine) importFuncs(funcs map[string]*api.Function) error {

	for k, v := range funcs {
		*v = f.module.ExportedFunction(k)
		if *v == nil {
			return fmt.Errorf("missing exported function: %s", k)
		}
	}
	return nil
}

func (f *wazeroExtEngine) init(ctx context.Context, config iextengine.ExtEngineConfig) error {
	var err error
	var memPages = config.MemoryLimitPages
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

	f.ctx = ctx
	rtConf := wazero.NewRuntimeConfigInterpreter().
		WithFeatureBulkMemoryOperations(true).
		//WithFeatureSignExtensionOps(true).
		//WithFeatureNonTrappingFloatToIntConversion(true).
		WithMemoryLimitPages(uint32(memPages))

	rtm := wazero.NewRuntimeWithConfig(rtConf)

	f.host, err = rtm.NewModuleBuilder("wasi_snapshot_preview1").
		ExportFunction("fd_write", f.fdWrite).
		Instantiate(ctx)
	if err != nil {
		return err
	}

	f.host, err = rtm.NewModuleBuilder("env").
		ExportFunction("hostGetKey", f.hostGetKey).
		ExportFunction("hostGetValue", f.hostMustExist).
		ExportFunction("hostQueryValue", f.hostCanExist).
		ExportFunction("hostReadValues", f.hostReadValues).
		ExportFunction("hostPanic", f.hostPanic).

		// IKey
		ExportFunction("hostKeyAsString", f.hostKeyAsString).
		ExportFunction("hostKeyAsBytes", f.hostKeyAsBytes).
		ExportFunction("hostKeyAsInt32", f.hostKeyAsInt32).
		ExportFunction("hostKeyAsInt64", f.hostKeyAsInt64).
		ExportFunction("hostKeyAsFloat32", f.hostKeyAsFloat32).
		ExportFunction("hostKeyAsFloat64", f.hostKeyAsFloat64).
		ExportFunction("hostKeyAsBool", f.hostKeyAsBool).
		ExportFunction("hostKeyAsQNamePkg", f.hostKeyAsQNamePkg).
		ExportFunction("hostKeyAsQNameEntity", f.hostKeyAsQNameEntity).

		// IValue
		ExportFunction("hostValueLength", f.hostValueLength).
		ExportFunction("hostValueAsValue", f.hostValueAsValue).
		ExportFunction("hostValueAsString", f.hostValueAsString).
		ExportFunction("hostValueAsBytes", f.hostValueAsBytes).
		ExportFunction("hostValueAsInt32", f.hostValueAsInt32).
		ExportFunction("hostValueAsInt64", f.hostValueAsInt64).
		ExportFunction("hostValueAsFloat32", f.hostValueAsFloat32).
		ExportFunction("hostValueAsFloat64", f.hostValueAsFloat64).
		ExportFunction("hostValueAsQNamePkg", f.hostValueAsQNamePkg).
		ExportFunction("hostValueAsQNameEntity", f.hostValueAsQNameEntity).
		ExportFunction("hostValueAsBool", f.hostValueAsBool).
		ExportFunction("hostValueGetAsBytes", f.hostValueGetAsBytes).
		ExportFunction("hostValueGetAsString", f.hostValueGetAsString).
		ExportFunction("hostValueGetAsInt32", f.hostValueGetAsInt32).
		ExportFunction("hostValueGetAsInt64", f.hostValueGetAsInt64).
		ExportFunction("hostValueGetAsFloat32", f.hostValueGetAsFloat32).
		ExportFunction("hostValueGetAsFloat64", f.hostValueGetAsFloat64).
		ExportFunction("hostValueGetAsValue", f.hostValueGetAsValue).
		ExportFunction("hostValueGetAsQNamePkg", f.hostValueGetAsQNamePkg).
		ExportFunction("hostValueGetAsQNameEntity", f.hostValueGetAsQNameEntity).
		ExportFunction("hostValueGetAsBool", f.hostValueGetAsBool).

		// Intents
		ExportFunction("hostNewValue", f.hostNewValue).
		ExportFunction("hostUpdateValue", f.hostUpdateValue).

		// RowWriters
		ExportFunction("hostRowWriterPutString", f.hostRowWriterPutString).
		ExportFunction("hostRowWriterPutBytes", f.hostRowWriterPutBytes).
		ExportFunction("hostRowWriterPutInt32", f.hostRowWriterPutInt32).
		ExportFunction("hostRowWriterPutInt64", f.hostRowWriterPutInt64).
		ExportFunction("hostRowWriterPutFloat32", f.hostRowWriterPutFloat32).
		ExportFunction("hostRowWriterPutFloat64", f.hostRowWriterPutFloat64).
		ExportFunction("hostRowWriterPutBool", f.hostRowWriterPutBool).
		ExportFunction("hostRowWriterPutQName", f.hostRowWriterPutQName).

		//ExportFunction("printstr", f.printStr).
		Instantiate(ctx)
	if err != nil {
		return err
	}

	f.module, err = rtm.InstantiateModuleFromCode(ctx, f.data)
	if err != nil {
		return err
	}

	err = f.importFuncs(map[string]*api.Function{
		"malloc":               &f.funcMalloc,
		"free":                 &f.funcFree,
		"WasmAbiVersion_0_0_1": &f.funcVer,
		"WasmGetHeapInuse":     &f.funcGetHeapInuse,
		"WasmGetHeapSys":       &f.funcGetHeapSys,
		"WasmGetMallocs":       &f.funcGetMallocs,
		"WasmGetFrees":         &f.funcGetFrees,
		"WasmGC":               &f.funcGc,
		"WasmOnReadValue":      &f.funcOnReadValue,
	})
	if err != nil {
		return err
	}

	f.ce = f.module.NewCallEngine()

	// Check WASM SDK version
	_, err = f.funcVer.CallEx(ctx, f.ce, nil)
	if err != nil {
		return errors.New("unsupported WASM version")
	}

	f.keyBuilders = make([]istructs.IStateKeyBuilder, 0, keysBuildersCapacity)
	f.values = make([]istructs.IStateValue, 0, valuesCapacity)
	f.valueBuilders = make([]istructs.IStateValueBuilder, 0, valueBuildersCapacity)

	res, err := f.funcMalloc.Call(f.ctx, uint64(WasmPreallocatedBufferSize))
	if err != nil {
		return err
	}
	f.allocatedBufs = append(f.allocatedBufs, &allocatedBuf{
		addr: uint32(res[0]),
		offs: 0,
		cap:  WasmPreallocatedBufferSize,
	})

	f.recoverMem = f.module.Memory().Backup()

	return nil

}

func (f *wazeroExtEngine) close() {
	if f.module != nil {
		f.module.Close(f.ctx)
	}
	if f.host != nil {
		f.host.Close(f.ctx)
	}
	if f.mwasi != nil {
		f.mwasi.Close(f.ctx)
	}
}

func (f *wazeroExtEngine) ForEach(callback func(name string, ext iextengine.IExtension)) {

	exports := f.module.Exports()
	for i := 0; i < len(*exports); i++ {
		name := (*exports)[i]
		if !strings.HasPrefix(name, "Wasm") && name != "alloc" && name != "free" &&
			name != "calloc" && name != "realloc" && name != "malloc" && name != "_start" && name != "memory" {
			callback(name, &wasmExtension{name, f.module.ExportedFunction(name), f})
		}
	}

}

func (f *wazeroExtEngine) recover() {
	f.module.Memory().Restore(f.recoverMem)
}

func (f *wazeroExtEngine) invoke(funct api.Function, io iextengine.IExtentionIO) (err error) {

	f.io = io

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
	for i := range f.allocatedBufs {
		f.allocatedBufs[i].offs = 0 // reuse pre-allocated memory
	}
	_, err = funct.CallEx(f.ctx, f.ce, &f.cep)

	if err != nil {
		f.recover()
	}

	return err
}

func (f *wazeroExtEngine) decodeStr(ptr, size uint32) string {
	if bytes, ok := f.module.Memory().Read(f.ctx, uint32(ptr), uint32(size)); ok {
		return string(bytes)
	}
	panic(ErrUnableToReadMemory)
}

func (f *wazeroExtEngine) fdWrite(fd, iovs, iovsCount, resultSize uint32) wasi.Errno {
	return wasi.ErrnoNosys
}

const thirdArgument = 3
const fourthArgument = 4
const fifthArgument = 5
const sixthArgument = 6
const seventhArgument = 7

func (f *wazeroExtEngine) hostGetKey(args []uint64) (res []uint64) {

	var storagePtr uint32 = uint32(args[0])
	var storageSize uint32 = uint32(args[1])
	var entityPtr uint32 = uint32(args[2])
	var entitySize uint32 = uint32(args[thirdArgument])

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
	res = []uint64{uint64(len(f.keyBuilders))}
	f.keyBuilders = append(f.keyBuilders, k)
	return
}

func (f *wazeroExtEngine) hostPanic(args []uint64) (result []uint64) {
	var namePtr uint32 = uint32(args[0])
	var nameSize uint32 = uint32(args[1])
	panic(f.decodeStr(namePtr, nameSize))
}

func (f *wazeroExtEngine) hostReadValues(args []uint64) (result []uint64) {
	var keyId uint64 = args[0]
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
		_, err = f.funcOnReadValue.Call(f.ctx, uint64(keyIndex), uint64(valueIndex))
		return err
	})
	if err != nil {
		panic(err.Error())
	}
	return []uint64{}
}

func (f *wazeroExtEngine) hostMustExist(args []uint64) (result []uint64) {
	var keyId uint64 = args[0]
	if int(keyId) >= len(f.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	v, e := f.io.MustExist(f.keyBuilders[keyId])
	if e != nil {
		panic(e)
	}
	result = []uint64{uint64(len(f.values))}
	f.values = append(f.values, v)
	return
}

const maxUint64 = ^uint64(0)

func (f *wazeroExtEngine) hostCanExist(args []uint64) (result []uint64) {
	var keyId uint64 = args[0]
	if int(keyId) >= len(f.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	v, ok, e := f.io.CanExist(f.keyBuilders[keyId])
	if e != nil {
		panic(e)
	}
	if !ok {
		return []uint64{maxUint64}
	}
	result = []uint64{uint64(len(f.values))}
	f.values = append(f.values, v)
	return
}

func (f *wazeroExtEngine) allocAndSend(buf []byte) (result []uint64) {
	addrPkg, e := f.allocBuf(uint32(len(buf)))
	if e != nil {
		panic(e)
	}
	if !f.module.Memory().Write(f.ctx, addrPkg, buf) {
		panic(e)
	}
	return []uint64{(uint64(addrPkg) << uint64(bitsInFourBytes)) | uint64(len(buf))}
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

func (f *wazeroExtEngine) hostKeyAsString(args []uint64) (result []uint64) {
	key, name := f.keyargs(args[0], uint32(args[1]), uint32(args[2]))
	return f.allocAndSend([]byte(key.AsString(name)))
}

func (f *wazeroExtEngine) hostKeyAsBytes(args []uint64) (result []uint64) {
	key, name := f.keyargs(args[0], uint32(args[1]), uint32(args[2]))
	return f.allocAndSend(key.AsBytes(name))
}

func (f *wazeroExtEngine) hostKeyAsInt32(args []uint64) (result []uint64) {
	key, name := f.keyargs(args[0], uint32(args[1]), uint32(args[2]))
	return []uint64{uint64(key.AsInt32(name))}
}

func (f *wazeroExtEngine) hostKeyAsInt64(args []uint64) (result []uint64) {
	key, name := f.keyargs(args[0], uint32(args[1]), uint32(args[2]))
	return []uint64{uint64(key.AsInt64(name))}
}

func (f *wazeroExtEngine) hostKeyAsBool(args []uint64) (result []uint64) {
	key, name := f.keyargs(args[0], uint32(args[1]), uint32(args[2]))
	if key.AsBool(name) {
		return []uint64{1}
	}
	return []uint64{0}
}

func (f *wazeroExtEngine) hostKeyAsQNamePkg(args []uint64) (result []uint64) {
	key, name := f.keyargs(args[0], uint32(args[1]), uint32(args[2]))
	qname := key.AsQName(name)
	return f.allocAndSend([]byte(qname.Pkg()))
}

func (f *wazeroExtEngine) hostKeyAsQNameEntity(args []uint64) (result []uint64) {
	key, name := f.keyargs(args[0], uint32(args[1]), uint32(args[2]))
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

func (f *wazeroExtEngine) hostValueGetAsString(args []uint64) (result []uint64) {
	v := f.value(args[0])
	return f.allocAndSend([]byte(v.GetAsString(int(args[1]))))
}

func (f *wazeroExtEngine) hostValueGetAsQNameEntity(args []uint64) (result []uint64) {
	v := f.value(args[0])
	qname := v.GetAsQName(int(args[1]))
	return f.allocAndSend([]byte(qname.Entity()))
}

func (f *wazeroExtEngine) hostValueGetAsQNamePkg(args []uint64) (result []uint64) {
	v := f.value(args[0])
	qname := v.GetAsQName(int(args[1]))
	return f.allocAndSend([]byte(qname.Pkg()))
}

func (f *wazeroExtEngine) hostValueGetAsBytes(args []uint64) (result []uint64) {
	v := f.value(args[0])
	return f.allocAndSend(v.GetAsBytes(int(args[1])))
}

func (f *wazeroExtEngine) hostValueGetAsBool(args []uint64) (result []uint64) {
	v := f.value(args[0])
	if v.GetAsBool(int(args[1])) {
		return []uint64{1}
	}
	return []uint64{0}
}

func (f *wazeroExtEngine) hostValueGetAsInt32(args []uint64) (result []uint64) {
	v := f.value(args[0])
	return []uint64{uint64(v.GetAsInt32(int(args[1])))}
}

func (f *wazeroExtEngine) hostValueGetAsInt64(args []uint64) (result []uint64) {
	v := f.value(args[0])
	return []uint64{uint64(v.GetAsInt64(int(args[1])))}
}

func (f *wazeroExtEngine) hostValueGetAsFloat32(id uint64, index uint32) float32 {
	return f.value(id).GetAsFloat32(int(index))
}

func (f *wazeroExtEngine) hostValueGetAsFloat64(id uint64, index uint32) float64 {
	return f.value(id).GetAsFloat64(int(index))
}

func (f *wazeroExtEngine) hostValueGetAsValue(args []uint64) (result []uint64) {
	v := f.value(args[0])
	value := v.GetAsValue(int(args[1]))
	result = []uint64{uint64(len(f.values))}
	f.values = append(f.values, value)
	return
}

func (f *wazeroExtEngine) hostValueAsString(args []uint64) (result []uint64) {
	v, name := f.valueargs(args[0], uint32(args[1]), uint32(args[2]))
	return f.allocAndSend([]byte(v.AsString(name)))
}

func (f *wazeroExtEngine) hostValueAsBytes(args []uint64) (result []uint64) {
	v, name := f.valueargs(args[0], uint32(args[1]), uint32(args[2]))
	return f.allocAndSend(v.AsBytes(name))
}

func (f *wazeroExtEngine) hostValueAsInt32(args []uint64) (result []uint64) {
	v, name := f.valueargs(args[0], uint32(args[1]), uint32(args[2]))
	return []uint64{uint64(v.AsInt32(name))}
}

func (f *wazeroExtEngine) hostValueAsInt64(args []uint64) (result []uint64) {
	v, name := f.valueargs(args[0], uint32(args[1]), uint32(args[2]))
	return []uint64{uint64(v.AsInt64(name))}
}

func (f *wazeroExtEngine) hostValueAsBool(args []uint64) (result []uint64) {
	v, name := f.valueargs(args[0], uint32(args[1]), uint32(args[2]))
	if v.AsBool(name) {
		return []uint64{uint64(1)}
	}
	return []uint64{uint64(0)}
}

func (f *wazeroExtEngine) hostValueAsFloat32(id uint64, namePtr, nameSize uint32) float32 {
	v, name := f.valueargs(id, namePtr, nameSize)
	return v.AsFloat32(name)
}

func (f *wazeroExtEngine) hostValueAsFloat64(id uint64, namePtr, nameSize uint32) float64 {
	v, name := f.valueargs(id, namePtr, nameSize)
	return v.AsFloat64(name)
}

func (f *wazeroExtEngine) hostValueAsQNameEntity(args []uint64) (result []uint64) {
	v, name := f.valueargs(args[0], uint32(args[1]), uint32(args[2]))
	qname := v.AsQName(name)
	return f.allocAndSend([]byte(qname.Entity()))
}

func (f *wazeroExtEngine) hostValueAsQNamePkg(args []uint64) (result []uint64) {
	v, name := f.valueargs(args[0], uint32(args[1]), uint32(args[2]))
	qname := v.AsQName(name)
	return f.allocAndSend([]byte(qname.Pkg()))
}

func (f *wazeroExtEngine) hostValueAsValue(args []uint64) (result []uint64) {
	v, name := f.valueargs(args[0], uint32(args[1]), uint32(args[2]))
	value := v.AsValue(name)
	result = []uint64{uint64(len(f.values))}
	f.values = append(f.values, value)
	return
}

func (f *wazeroExtEngine) hostValueLength(args []uint64) (result []uint64) {
	var id uint64 = args[0]
	if int(id) >= len(f.values) {
		panic(PanicIncorrectValue)
	}
	return []uint64{uint64(f.values[id].Length())}
}

func (f *wazeroExtEngine) allocBuf(size uint32) (addr uint32, err error) {
	for i := range f.allocatedBufs {
		if f.allocatedBufs[i].cap-f.allocatedBufs[i].offs >= size {
			addr = f.allocatedBufs[i].addr + f.allocatedBufs[i].offs
			f.allocatedBufs[i].offs += uint32(size)
			return
		}
	}
	// no space in the allocated buffers

	var newBufferSize uint32 = WasmPreallocatedBufferIncrease
	if size > newBufferSize {
		newBufferSize = size
	}

	var res []uint64
	res, err = f.funcMalloc.CallEx(f.ctx, f.ce, &f.cep, uint64(newBufferSize))
	if err != nil {
		return 0, err
	}
	addr = uint32(res[0])
	f.allocatedBufs = append(f.allocatedBufs, &allocatedBuf{
		addr: addr,
		offs: 0,
		cap:  newBufferSize,
	})
	return addr, nil
}

func (f *wazeroExtEngine) getFrees() (uint64, error) {
	res, err := f.funcGetFrees.Call(f.ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) gc() error {
	_, err := f.funcGc.Call(f.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (f *wazeroExtEngine) getHeapinuse() (uint64, error) {
	res, err := f.funcGetHeapInuse.Call(f.ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) getHeapSys() (uint64, error) {
	res, err := f.funcGetHeapSys.Call(f.ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) getMallocs() (uint64, error) {
	res, err := f.funcGetMallocs.Call(f.ctx)
	if err != nil {
		return 0, err
	}
	return res[0], nil
}

func (f *wazeroExtEngine) hostNewValue(args []uint64) (result []uint64) {
	var keyId uint64 = args[0]
	if int(keyId) >= len(f.keyBuilders) {
		panic(PanicIncorrectKeyBuilder)
	}
	vb, err := f.io.NewValue(f.keyBuilders[keyId])
	if err != nil {
		panic(err)
	}
	result = []uint64{uint64(len(f.valueBuilders))}
	f.valueBuilders = append(f.valueBuilders, vb)
	return
}

func (f *wazeroExtEngine) hostUpdateValue(args []uint64) (result []uint64) {
	var keyId uint64 = args[0]
	var existingValueId uint64 = args[1]
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
	result = []uint64{uint64(len(f.valueBuilders))}
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

func (f *wazeroExtEngine) hostRowWriterPutString(args []uint64) []uint64 {
	writer, name := f.getWriterArgs(args[0], uint32(args[1]), uint32(args[2]), uint32(args[thirdArgument]))
	var valuePtr uint32 = uint32(args[fourthArgument])
	var valueSize uint32 = uint32(args[fifthArgument])
	writer.PutString(name, f.decodeStr(valuePtr, valueSize))
	return []uint64{}
}

func (f *wazeroExtEngine) hostRowWriterPutBytes(args []uint64) []uint64 {
	writer, name := f.getWriterArgs(args[0], uint32(args[1]), uint32(args[2]), uint32(args[thirdArgument]))
	var valuePtr uint32 = uint32(args[fourthArgument])
	var valueSize uint32 = uint32(args[fifthArgument])

	var bytes []byte
	var ok bool
	bytes, ok = f.module.Memory().Read(f.ctx, uint32(valuePtr), uint32(valueSize))
	if !ok {
		panic(ErrUnableToReadMemory)
	}

	writer.PutBytes(name, bytes)
	return []uint64{}
}

func (f *wazeroExtEngine) hostRowWriterPutInt32(args []uint64) []uint64 {
	writer, name := f.getWriterArgs(args[0], uint32(args[1]), uint32(args[2]), uint32(args[thirdArgument]))
	writer.PutInt32(name, int32(args[fourthArgument]))
	return []uint64{}
}

func (f *wazeroExtEngine) hostRowWriterPutInt64(args []uint64) []uint64 {
	writer, name := f.getWriterArgs(args[0], uint32(args[1]), uint32(args[2]), uint32(args[thirdArgument]))
	writer.PutInt64(name, int64(args[fourthArgument]))
	return []uint64{}
}

func (f *wazeroExtEngine) hostRowWriterPutQName(args []uint64) []uint64 {
	writer, name := f.getWriterArgs(args[0], uint32(args[1]), uint32(args[2]), uint32(args[thirdArgument]))
	pkg := f.decodeStr(uint32(args[fourthArgument]), uint32(args[fifthArgument]))
	entity := f.decodeStr(uint32(args[sixthArgument]), uint32(args[seventhArgument]))
	writer.PutQName(name, appdef.NewQName(pkg, entity))
	return []uint64{}
}

func (f *wazeroExtEngine) hostRowWriterPutBool(args []uint64) []uint64 {
	writer, name := f.getWriterArgs(args[0], uint32(args[1]), uint32(args[2]), uint32(args[thirdArgument]))
	writer.PutBool(name, int32(args[fourthArgument]) > 0)
	return []uint64{}
}

func (f *wazeroExtEngine) hostRowWriterPutFloat32(id uint64, typ uint32, namePtr, nameSize uint32, value float32) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)
	writer.PutFloat32(name, value)
}

func (f *wazeroExtEngine) hostRowWriterPutFloat64(id uint64, typ uint32, namePtr, nameSize uint32, value float64) {
	writer, name := f.getWriterArgs(id, typ, namePtr, nameSize)
	writer.PutFloat64(name, value)
}
