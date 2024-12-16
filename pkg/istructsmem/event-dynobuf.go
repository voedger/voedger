/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"io"
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
)

func storeEvent(ev *eventType, buf *bytes.Buffer) {
	utils.WriteUint16(buf, ev.qNameID())

	storeEventCreateParams(ev, buf)
	storeEventBuildError(ev, buf)

	if !ev.valid() {
		return
	}

	storeEventArguments(ev, buf)
	storeEventCUDs(ev, buf)
}

func storeEventCreateParams(ev *eventType, buf *bytes.Buffer) {
	utils.WriteUint16(buf, uint16(ev.partition))
	utils.WriteUint64(buf, uint64(ev.pLogOffs))
	utils.WriteUint64(buf, uint64(ev.ws))
	utils.WriteUint64(buf, uint64(ev.wLogOffs))
	utils.WriteInt64(buf, int64(ev.regTime))
	utils.WriteBool(buf, ev.sync)
	if ev.sync {
		utils.WriteUint16(buf, uint16(ev.device))
		utils.WriteInt64(buf, int64(ev.syncTime))
	}
}

func storeEventBuildError(ev *eventType, buf *bytes.Buffer) {

	valid := ev.valid()
	utils.WriteBool(buf, valid)

	if valid {
		return
	}

	utils.WriteShortString(buf, ev.buildErr.ErrStr())

	utils.WriteShortString(buf, ev.name.String())

	bytes := ev.buildErr.OriginalEventBytes()
	bytesLen := uint32(len(bytes)) // nolint G115

	if ev.argUnlObj.QName() != appdef.NullQName {
		bytesLen = 0 // to protect logging security data
	}

	utils.WriteUint32(buf, bytesLen)

	if bytesLen > 0 {
		utils.SafeWriteBuf(buf, bytes)
	}
}

func storeEventArguments(ev *eventType, buf *bytes.Buffer) {
	storeObject(&ev.argObject, buf)
	storeObject(&ev.argUnlObj, buf)
}

func storeEventCUDs(ev *eventType, buf *bytes.Buffer) {
	count := uint16(len(ev.cud.creates)) // nolint G115 validated in [validateEventCUDs]
	utils.WriteUint16(buf, count)
	for _, rec := range ev.cud.creates {
		storeEventCUD(rec, buf)
	}

	count = uint16(len(ev.cud.updates)) // nolint G115 validated in [validateEventCUDs]
	utils.WriteUint16(buf, count)
	for _, rec := range ev.cud.updates {
		storeEventCUD(&rec.changes, buf)
	}
}

func storeEventCUD(rec *recordType, buf *bytes.Buffer) {
	storeRow(&rec.rowType, buf)

	// #2785: store emptied fields
	emptied := make([]uint16, 0, len(rec.nils))
	if len(rec.nils) > 0 {
		fields := rec.fields.UserFields()
		for _, f := range rec.nils {
			if idx := slices.Index(fields, f); idx >= 0 {
				emptied = append(emptied, uint16(idx)) // nolint G115 see [appdef.MaxTypeFieldCount]
			}
		}
	}
	utils.WriteUint16(buf, uint16(len(emptied))) // nolint G115 see [appdef.MaxTypeFieldCount]
	for i := range emptied {
		utils.WriteUint16(buf, emptied[i])
	}
}

func storeObject(o *objectType, buf *bytes.Buffer) {

	storeRow(&o.rowType, buf)

	if o.QName() == appdef.NullQName {
		return
	}

	childCount := uint16(len(o.child)) // nolint G115 validated, see [objectType.build]
	utils.WriteUint16(buf, childCount)
	for _, c := range o.child {
		storeObject(c, buf)
	}
}

func loadEvent(ev *eventType, codecVer byte, buf *bytes.Buffer) (err error) {
	var id uint16
	if id, err = utils.ReadUInt16(buf); err != nil {
		return enrichError(err, "event QName ID")
	}
	if ev.name, err = ev.appCfg.qNames.QName(id); err != nil {
		return enrichError(err, "event QName")
	}

	if ev.name == appdef.NullQName {
		return nil
	}

	if err := loadEventCreateParams(ev, buf); err != nil {
		return enrichError(err, "%v create params", ev.name)
	}

	if err := loadEventBuildError(ev, buf); err != nil {
		return enrichError(err, "%v build error", ev.name)
	}
	if !ev.valid() {
		return nil
	}

	if err := loadEventArguments(ev, codecVer, buf); err != nil {
		return enrichError(err, "%v arguments", ev.name)
	}

	if err := loadEventCUDs(ev, codecVer, buf); err != nil {
		return enrichError(err, "%v CUDs", ev.name)
	}

	return nil
}

func loadEventCreateParams(ev *eventType, buf *bytes.Buffer) (err error) {
	if p, err := utils.ReadUInt16(buf); err == nil {
		ev.partition = istructs.PartitionID(p)
	} else {
		return enrichError(err, "partition id")
	}

	if o, err := utils.ReadUInt64(buf); err == nil {
		ev.pLogOffs = istructs.Offset(o)
	} else {
		return enrichError(err, "PLog offset")
	}

	if w, err := utils.ReadUInt64(buf); err == nil {
		ev.ws = istructs.WSID(w)
	} else {
		return enrichError(err, "workspace id")
	}

	if o, err := utils.ReadUInt64(buf); err == nil {
		ev.wLogOffs = istructs.Offset(o)
	} else {
		return enrichError(err, "WLog offset")
	}

	if t, err := utils.ReadInt64(buf); err == nil {
		ev.regTime = istructs.UnixMilli(t)
	} else {
		return enrichError(err, "register time")
	}

	if ev.sync, err = utils.ReadBool(buf); err != nil {
		return enrichError(err, "synch flag")
	}

	if ev.sync {
		if d, err := utils.ReadUInt16(buf); err == nil {
			ev.device = istructs.ConnectedDeviceID(d)
		} else {
			return enrichError(err, "device ID")
		}

		if t, err := utils.ReadInt64(buf); err == nil {
			ev.syncTime = istructs.UnixMilli(t)
		} else {
			return enrichError(err, "synch time")
		}
	}

	return nil
}

func loadEventBuildError(ev *eventType, buf *bytes.Buffer) (err error) {
	if ev.buildErr.validEvent, err = utils.ReadBool(buf); err != nil {
		return enrichError(err, "validation result")
	}

	if ev.buildErr.validEvent {
		return nil
	}

	if ev.buildErr.errStr, err = utils.ReadShortString(buf); err != nil {
		return enrichError(err, "build error message")
	}

	qName := ""
	if qName, err = utils.ReadShortString(buf); err != nil {
		return enrichError(err, "original event name")
	}
	if ev.buildErr.qName, err = appdef.ParseQName(qName); err != nil {
		return enrichError(err, "original event name")
	}

	bytesLen := uint32(0)
	if bytesLen, err = utils.ReadUInt32(buf); err != nil {
		return enrichError(err, "source raw bytes length")
	}

	if buf.Len() < int(bytesLen) {
		return enrichError(io.ErrUnexpectedEOF, "source raw bytes, expected %d bytes, but only %d bytes is available", bytesLen, buf.Len())
	}

	ev.buildErr.bytes = make([]byte, bytesLen)
	if _, err = buf.Read(ev.buildErr.bytes); err != nil {
		// no test: possible error (only EOF) is handled above
		return enrichError(err, "source raw bytes")
	}

	return nil
}

func loadEventArguments(ev *eventType, codecVer byte, buf *bytes.Buffer) (err error) {
	if err := loadObject(&ev.argObject, codecVer, buf); err != nil {
		return enrichError(err, "argument")
	}

	if err := loadObject(&ev.argUnlObj, codecVer, buf); err != nil {
		return enrichError(err, "unlogged argument")
	}

	return nil
}

func loadEventCUDs(ev *eventType, codecVer byte, buf *bytes.Buffer) (err error) {
	count := uint16(0)
	if count, err = utils.ReadUInt16(buf); err != nil {
		return enrichError(err, "CUDs new rows count")
	}
	for ; count > 0; count-- {
		rec := newRecord(ev.cud.appCfg)
		rec.isNew = true
		if err := loadEventCUD(rec, codecVer, buf); err != nil {
			return enrichError(err, "CUD new row")
		}
		ev.cud.creates = append(ev.cud.creates, rec)
	}

	count = uint16(0)
	if count, err = utils.ReadUInt16(buf); err != nil {
		return enrichError(err, "CUDs updated rows count")
	}
	for ; count > 0; count-- {
		upd := newUpdateRec(ev.cud.appCfg, newRecord(ev.cud.appCfg))
		if err := loadEventCUD(&upd.changes, codecVer, buf); err != nil {
			return enrichError(err, "CUD updated row")
		}
		id := upd.changes.ID()
		upd.originRec.setQName(upd.changes.QName())
		upd.originRec.setID(id)
		// ⚠ Warnings:
		// — upd.originRec is partially constructed, not full filled!
		// — upd.result is null record, not applicable to store!
		// it is very important for calling code to reread upd.originRec and recall upd.build() to obtain correct upd.result
		ev.cud.updates[id] = &upd
	}

	return nil
}

func loadEventCUD(rec *recordType, codecVer byte, buf *bytes.Buffer) error {
	if err := loadRow(&rec.rowType, codecVer, buf); err != nil {
		return enrichError(err, "CUD row")
	}
	// #2785: read emptied fields
	if codecVer >= codec_RDB_2 {
		count, err := utils.ReadUInt16(buf)
		if err != nil {
			return enrichError(err, "emptied field count")
		}
		if toRead := int(count) * uint16len; toRead > buf.Len() {
			return enrichError(io.ErrUnexpectedEOF, "emptied fields indexes, expected %d bytes, but only %d bytes is available", toRead, buf.Len())
		}
		fields := rec.fields.UserFields()
		len := uint16(len(fields)) // nolint G115 see [appdef.MaxTypeFieldCount]
		for i := uint16(0); i < count; i++ {
			idx, err := utils.ReadUInt16(buf)
			if err != nil {
				// no test: possible error (only EOF) is handled above
				return enrichError(err, "emptied field[%d] index", i)
			}
			if idx >= len {
				return ErrOutOfBounds("emptied field[%d] index %d should be less than %d", i, idx, len)
			}
			f := fields[idx]
			if k := f.DataKind(); k != appdef.DataKind_string && k != appdef.DataKind_bytes {
				return ErrWrongType("emptied %v should be string- (or []byte-) field", f)
			}
			rec.checkPutNil(f, nil)
		}
	}
	return nil
}

func loadObject(o *objectType, codecVer byte, buf *bytes.Buffer) (err error) {

	if err := loadRow(&o.rowType, codecVer, buf); err != nil {
		return err
	}

	if o.QName() == appdef.NullQName {
		return nil
	}

	count := uint16(0)
	if count, err = utils.ReadUInt16(buf); err != nil {
		return enrichError(err, "child count for %v", o)
	}
	for i := uint16(0); i < count; i++ {
		child := newObject(o.appCfg, appdef.NullQName, o)
		if err := loadObject(child, codecVer, buf); err != nil {
			return enrichError(err, "%v child[%d]", o, i)
		}
		o.child = append(o.child, child)
	}

	return nil
}
