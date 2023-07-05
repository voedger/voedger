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

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
)

func storeEvent(ev *eventType, buf *bytes.Buffer) {
	utils.SafeWriteBuf(buf, uint16(ev.qNameID()))

	storeEventCreateParams(ev, buf)
	storeEventBuildError(ev, buf)

	if !ev.valid() {
		return
	}

	storeEventArguments(ev, buf)
	storeEventCUDs(ev, buf)
}

func storeEventCreateParams(ev *eventType, buf *bytes.Buffer) {
	utils.SafeWriteBuf(buf, ev.partition)
	utils.SafeWriteBuf(buf, ev.pLogOffs)
	utils.SafeWriteBuf(buf, ev.ws)
	utils.SafeWriteBuf(buf, ev.wLogOffs)
	utils.SafeWriteBuf(buf, ev.regTime)
	utils.SafeWriteBuf(buf, ev.sync)
	if ev.sync {
		utils.SafeWriteBuf(buf, ev.device)
		utils.SafeWriteBuf(buf, ev.syncTime)
	}
}

func storeEventBuildError(ev *eventType, buf *bytes.Buffer) {

	valid := ev.valid()
	utils.SafeWriteBuf(buf, valid)

	if valid {
		return
	}

	utils.WriteShortString(buf, ev.buildErr.ErrStr())

	utils.WriteShortString(buf, ev.name.String())

	bytes := ev.buildErr.OriginalEventBytes()
	bytesLen := uint32(len(bytes))

	if ev.argUnlObj.QName() != appdef.NullQName {
		bytesLen = 0 // to protect logging security data
	}

	utils.SafeWriteBuf(buf, bytesLen)

	if bytesLen > 0 {
		utils.SafeWriteBuf(buf, bytes)
	}
}

func storeEventArguments(ev *eventType, buf *bytes.Buffer) {
	storeElement(&ev.argObject, buf)
	storeElement(&ev.argUnlObj, buf)
}

func storeEventCUDs(ev *eventType, buf *bytes.Buffer) {
	count := uint16(len(ev.cud.creates))
	utils.SafeWriteBuf(buf, count)
	for _, rec := range ev.cud.creates {
		storeRow(&rec.rowType, buf)
	}

	count = uint16(len(ev.cud.updates))
	utils.SafeWriteBuf(buf, count)
	for _, rec := range ev.cud.updates {
		storeRow(&rec.changes.rowType, buf)
	}
}

func storeElement(el *elementType, buf *bytes.Buffer) {

	storeRow(&el.rowType, buf)

	if el.QName() == appdef.NullQName {
		return
	}

	childCount := uint16(len(el.child))
	utils.SafeWriteBuf(buf, childCount)
	for _, c := range el.child {
		storeElement(c, buf)
	}
}

func loadEvent(ev *eventType, codecVer byte, buf *bytes.Buffer) (err error) {
	var id uint16
	if err := binary.Read(buf, binary.BigEndian, &id); err != nil {
		return fmt.Errorf("error read event name ID: %w", err)
	}
	if ev.name, err = ev.appCfg.qNames.QName(qnames.QNameID(id)); err != nil {
		return fmt.Errorf("error read event name: %w", err)
	}

	if ev.name == appdef.NullQName {
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

func loadEventCreateParams(ev *eventType, buf *bytes.Buffer) (err error) {
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

func loadEventBuildError(ev *eventType, buf *bytes.Buffer) (err error) {
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
	if ev.buildErr.qName, err = appdef.ParseQName(qName); err != nil {
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

func loadEventArguments(ev *eventType, codecVer byte, buf *bytes.Buffer) (err error) {
	if err := loadElement(&ev.argObject, codecVer, buf); err != nil {
		return fmt.Errorf("can not load event command «%v» argument: %w", ev.name, err)
	}

	if err := loadElement(&ev.argUnlObj, codecVer, buf); err != nil {
		return fmt.Errorf("can not load event command «%v» un-logged argument: %w", ev.name, err)
	}

	return nil
}

func loadEventCUDs(ev *eventType, codecVer byte, buf *bytes.Buffer) (err error) {
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
		// — upd.originRec is partially constructed, not full filled!
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

	if el.QName() == appdef.NullQName {
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
		el.child = append(el.child, &child)
	}

	return nil
}
