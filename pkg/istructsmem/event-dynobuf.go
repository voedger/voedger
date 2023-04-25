/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/schemas"
)

func storeEvent(ev *dbEventType, buf *bytes.Buffer) (err error) {
	id := ev.qNameID()
	_ = binary.Write(buf, binary.BigEndian, uint16(id))

	storeEventCreateParams(ev, buf)
	storeEventBuildError(ev, buf)

	if !ev.valid() {
		return nil
	}

	if err := storeEventArguments(ev, buf); err != nil {
		return err
	}

	if err := storeEventCUDs(ev, buf); err != nil {
		return err
	}

	return nil
}

func storeEventCreateParams(ev *dbEventType, buf *bytes.Buffer) {
	_ = binary.Write(buf, binary.BigEndian, &ev.partition)
	_ = binary.Write(buf, binary.BigEndian, &ev.pLogOffs)
	_ = binary.Write(buf, binary.BigEndian, &ev.ws)
	_ = binary.Write(buf, binary.BigEndian, &ev.wLogOffs)
	_ = binary.Write(buf, binary.BigEndian, &ev.regTime)
	_ = binary.Write(buf, binary.BigEndian, &ev.sync)
	if ev.sync {
		_ = binary.Write(buf, binary.BigEndian, &ev.device)
		_ = binary.Write(buf, binary.BigEndian, &ev.syncTime)
	}
}

func storeEventBuildError(ev *dbEventType, buf *bytes.Buffer) {

	valid := ev.valid()
	_ = binary.Write(buf, binary.BigEndian, &valid)

	if valid {
		return
	}

	utils.WriteShortString(buf, ev.buildErr.ErrStr())

	utils.WriteShortString(buf, ev.name.String())

	bytes := ev.buildErr.OriginalEventBytes()
	bytesLen := uint32(len(bytes))

	if ev.argUnlObj.QName() != schemas.NullQName {
		bytesLen = 0 // to protect logging security data
	}

	_ = binary.Write(buf, binary.BigEndian, &bytesLen)

	if bytesLen > 0 {
		_, _ = buf.Write(bytes)
	}
}

func storeEventArguments(ev *dbEventType, buf *bytes.Buffer) (err error) {

	if err := storeElement(&ev.argObject, buf); err != nil {
		return fmt.Errorf("can not store event command «%v» argument «%v»: %w", ev.name, ev.argObject.QName(), err)
	}

	if err := storeElement(&ev.argUnlObj, buf); err != nil {
		return fmt.Errorf("can not store event command «%v» unlogged argument «%v»: %w", ev.name, ev.argUnlObj.QName(), err)
	}

	return nil
}

func storeEventCUDs(ev *dbEventType, buf *bytes.Buffer) (err error) {
	count := uint16(len(ev.cud.creates))
	_ = binary.Write(buf, binary.BigEndian, &count)
	for _, rec := range ev.cud.creates {
		if err = storeRow(&rec.rowType, buf); err != nil {
			return fmt.Errorf("error write event cud.create() record: %w", err)
		}
	}

	count = uint16(len(ev.cud.updates))
	_ = binary.Write(buf, binary.BigEndian, &count)
	for _, rec := range ev.cud.updates {
		if err = storeRow(&rec.changes.rowType, buf); err != nil {
			return fmt.Errorf("error write event cud.update() record: %w", err)
		}
	}

	return nil
}

func storeElement(el *elementType, buf *bytes.Buffer) (err error) {

	if err := storeRow(&el.rowType, buf); err != nil {
		return err
	}

	if el.QName() == schemas.NullQName {
		return nil
	}

	childCount := uint16(len(el.childs))
	_ = binary.Write(buf, binary.BigEndian, &childCount)
	for _, c := range el.childs {
		if err := storeElement(c, buf); err != nil {
			return err
		}
	}

	return nil
}

func loadEvent(ev *dbEventType, codecVer byte, buf *bytes.Buffer) (err error) {
	var id uint16
	if err := binary.Read(buf, binary.BigEndian, &id); err != nil {
		return fmt.Errorf("error read event name ID: %w", err)
	}
	if ev.name, err = ev.appCfg.qNames.GetQName(qnames.QNameID(id)); err != nil {
		return fmt.Errorf("error read event name: %w", err)
	}

	if ev.name == schemas.NullQName {
		return nil
	}

	if err := loadEventCreateParams(ev, buf); err != nil {
		return err
	}

	if err := loadEventBuildError(ev, buf); err != nil {
		return err
	}
	if !ev.valid() {
		return nil
	}

	if err := loadEventArguments(ev, codecVer, buf); err != nil {
		return err
	}

	if err := loadEventCUDs(ev, codecVer, buf); err != nil {
		return err
	}

	return nil
}

func loadEventCreateParams(ev *dbEventType, buf *bytes.Buffer) (err error) {
	if err := binary.Read(buf, binary.BigEndian, &ev.partition); err != nil {
		return fmt.Errorf("error read event partition: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &ev.pLogOffs); err != nil {
		return fmt.Errorf("error read event PLog offset: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &ev.ws); err != nil {
		return fmt.Errorf("error read event workspace: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &ev.wLogOffs); err != nil {
		return fmt.Errorf("error read event WLog offset: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &ev.regTime); err != nil {
		return fmt.Errorf("error read event register time: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &ev.sync); err != nil {
		return fmt.Errorf("error read event synch flag: %w", err)
	}
	if ev.sync {
		if err := binary.Read(buf, binary.BigEndian, &ev.device); err != nil {
			return fmt.Errorf("error read event device ID: %w", err)
		}
		if err := binary.Read(buf, binary.BigEndian, &ev.syncTime); err != nil {
			return fmt.Errorf("error read event synch time: %w", err)
		}
	}

	return nil
}

func loadEventBuildError(ev *dbEventType, buf *bytes.Buffer) (err error) {
	if err := binary.Read(buf, binary.BigEndian, &ev.buildErr.validEvent); err != nil {
		return fmt.Errorf("error read event validation result: %w", err)
	}

	if ev.buildErr.validEvent {
		return nil
	}

	if ev.buildErr.errStr, err = utils.ReadShortString(buf); err != nil {
		return fmt.Errorf("error read build error message: %w", err)
	}

	qName := ""
	if qName, err = utils.ReadShortString(buf); err != nil {
		return fmt.Errorf("error read original event name: %w", err)
	}
	if ev.buildErr.qName, err = schemas.ParseQName(qName); err != nil {
		return fmt.Errorf("error read original event name: %w", err)
	}

	bytesLen := uint32(0)
	if err := binary.Read(buf, binary.BigEndian, &bytesLen); err != nil {
		return fmt.Errorf("error read event source raw bytes length: %w", err)
	}
	ev.buildErr.bytes = make([]byte, bytesLen)
	var len int
	if len, err = buf.Read(ev.buildErr.bytes); err != nil {
		return fmt.Errorf("error read event source raw bytes: %w", err)
	}
	if len < int(bytesLen) {
		return fmt.Errorf("error read event source raw bytes, expected %d bytes, but only %d bytes is available: %w", len, buf.Len(), io.ErrUnexpectedEOF)
	}

	return nil
}

func loadEventArguments(ev *dbEventType, codecVer byte, buf *bytes.Buffer) (err error) {
	if err := loadElement(&ev.argObject, codecVer, buf); err != nil {
		return fmt.Errorf("can not load event command «%v» argument: %w", ev.name, err)
	}

	if err := loadElement(&ev.argUnlObj, codecVer, buf); err != nil {
		return fmt.Errorf("can not load event command «%v» unlogged argument: %w", ev.name, err)
	}

	return nil
}

func loadEventCUDs(ev *dbEventType, codecVer byte, buf *bytes.Buffer) (err error) {
	count := uint16(0)
	if err := binary.Read(buf, binary.BigEndian, &count); err != nil {
		return fmt.Errorf("error read event cud.create() count: %w", err)
	}
	for ; count > 0; count-- {
		r := newRecord(ev.cud.appCfg)
		r.isNew = true
		rec := &r
		if err := loadRow(&rec.rowType, codecVer, buf); err != nil {
			return fmt.Errorf("error read event cud.create() record: %w", err)
		}
		ev.cud.creates = append(ev.cud.creates, rec)
	}

	count = uint16(0)
	if err := binary.Read(buf, binary.BigEndian, &count); err != nil {
		return fmt.Errorf("error read event cud.update() count: %w", err)
	}
	for ; count > 0; count-- {
		r := newRecord(ev.cud.appCfg)
		upd := newUpdateRec(ev.cud.appCfg, &r)
		if err := loadRow(&upd.changes.rowType, codecVer, buf); err != nil {
			return fmt.Errorf("error read event cud.update() record: %w", err)
		}
		id := upd.changes.ID()
		upd.originRec.setQName(upd.changes.QName())
		upd.originRec.setID(id)
		// warnings:
		// — upd.originRec is partially constructed, not full readed!
		// — upd.result is null record, not applicable to store!
		// it is very important for calling code to reread upd.originRec and recall upd.build() to obtain correct upd.result
		ev.cud.updates[id] = &upd
	}

	return nil
}

func loadElement(el *elementType, codecVer byte, buf *bytes.Buffer) (err error) {

	if err := loadRow(&el.rowType, codecVer, buf); err != nil {
		return err
	}

	if el.QName() == schemas.NullQName {
		return nil
	}

	childCount := uint16(0)
	if err := binary.Read(buf, binary.BigEndian, &childCount); err != nil {
		return err
	}
	for ; childCount > 0; childCount-- {
		child := newElement(el)
		if err := loadElement(&child, codecVer, buf); err != nil {
			return err
		}
		el.childs = append(el.childs, &child)
	}

	return nil
}
