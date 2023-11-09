/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package collection

import (
	"fmt"

	"github.com/untillpro/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
)

// Implements IElement
type collectionElement struct {
	istructs.IRecord
	elements   []*collectionElement
	rawRecords []istructs.IRecord
}

func newCollectionElement(rec istructs.IRecord) collectionElement {
	return collectionElement{
		IRecord:  rec,
		elements: make([]*collectionElement, 0),
	}
}

func (me *collectionElement) addElementsForParent(list []istructs.IRecord, parent istructs.RecordID) {
	for _, r := range list {
		if r.Parent() == parent {
			element := newCollectionElement(r)
			me.elements = append(me.elements, &element)
			if logger.IsVerbose() {
				logger.Verbose(fmt.Sprintf("collectionElem ID: %d: added ID: %d, QName: %s", me.ID(), r.ID(), r.QName().String()))
			}
			element.addElementsForParent(list, r.ID())
		}
	}
}

func (me *collectionElement) handleRawRecords() {
	me.addElementsForParent(me.rawRecords, me.ID())
}

func (me *collectionElement) addRawRecord(rec istructs.IRecord) {
	if me.rawRecords == nil {
		me.rawRecords = make([]istructs.IRecord, 1)
		me.rawRecords[0] = rec
	} else {
		me.rawRecords = append(me.rawRecords, rec)
	}
}

// Children in given container
func (me *collectionElement) Children(container string, cb func(el istructs.IElement)) {
	for i := range me.elements {
		el := me.elements[i]
		if el.Container() == container {
			cb(el)
		}
	}
}

// First level qname-s
func (me *collectionElement) Containers(cb func(container string)) {
	iterated := make(map[string]bool)
	for i := range me.elements {
		el := me.elements[i]
		eCont := el.Container()
		if _, ok := iterated[eCont]; !ok {
			iterated[eCont] = true
			cb(eCont)
		}
	}
}

func (me *collectionElement) AsRecord() istructs.IRecord {
	return me
}
