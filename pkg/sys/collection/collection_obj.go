/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package collection

import (
	"fmt"

	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
)

// Implements IElement
type collectionObject struct {
	istructs.IRecord
	children   []*collectionObject
	rawRecords []istructs.IRecord
}

func newCollectionObject(rec istructs.IRecord) *collectionObject {
	return &collectionObject{
		IRecord:  rec,
		children: make([]*collectionObject, 0),
	}
}

func (me *collectionObject) addElementsForParent(list []istructs.IRecord, parent istructs.RecordID) {
	for _, r := range list {
		if r.Parent() == parent {
			child := newCollectionObject(r)
			me.children = append(me.children, child)
			if logger.IsVerbose() {
				logger.Verbose(fmt.Sprintf("collectionElem ID: %d: added ID: %d, QName: %s", me.ID(), r.ID(), r.QName().String()))
			}
			child.addElementsForParent(list, r.ID())
		}
	}
}

func (me *collectionObject) handleRawRecords() {
	me.addElementsForParent(me.rawRecords, me.ID())
}

func (me *collectionObject) addRawRecord(rec istructs.IRecord) {
	if me.rawRecords == nil {
		me.rawRecords = make([]istructs.IRecord, 1)
		me.rawRecords[0] = rec
	} else {
		me.rawRecords = append(me.rawRecords, rec)
	}
}

// Children in given container
func (me *collectionObject) Children(container ...string) func(func(istructs.IObject) bool) {
	cc := make(map[string]bool)
	for _, c := range container {
		cc[c] = true
	}
	return func(cb func(istructs.IObject) bool) {
		for i := range me.children {
			c := me.children[i]
			if len(cc) == 0 || cc[c.Container()] {
				if !cb(c) {
					break
				}
			}
		}
	}
}

// First level qname-s
func (me *collectionObject) Containers(cb func(string) bool) {
	iterated := make(map[string]bool)
	for i := range me.children {
		c := me.children[i]
		cont := c.Container()
		if _, ok := iterated[cont]; !ok {
			iterated[cont] = true
			if !cb(cont) {
				break
			}
		}
	}
}

func (me *collectionObject) AsRecord() istructs.IRecord {
	return me
}
