/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/untillpro/dynobuffers"
	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorageimpl"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

var NullAppConfig = newAppConfig(istructs.AppQName_null, appdef.New())

var (
	nullDynoBuffer = dynobuffers.NewBuffer(dynobuffers.NewScheme())
	// not a func -> golang itokensjwt.TimeFunc will be initialized on process init forever
	testTokensFactory     = func() payloads.IAppTokensFactory { return payloads.TestAppTokensFactory(itokensjwt.TestTokensJWT()) }
	simpleStorageProvider = func() istorage.IAppStorageProvider {
		asf := istorage.ProvideMem()
		return istorageimpl.Provide(asf)
	}
)

// crackID splits ID to two-parts key — partition key (hi) and clustering columns (lo)
func crackID(id uint64) (hi uint64, low uint16) {
	return uint64(id >> partitionBits), uint16(id) & lowMask
}

// crackRecordID splits record ID to two-parts key — partition key (hi) and clustering columns (lo)
func crackRecordID(id istructs.RecordID) (hi uint64, low uint16) {
	return crackID(uint64(id))
}

// crackLogOffset splits log offset to two-parts key — partition key (hi) and clustering columns (lo)
func crackLogOffset(ofs istructs.Offset) (hi uint64, low uint16) {
	return crackID(uint64(ofs))
}

// glueLogOffset calculate log offset from two-parts key — partition key (hi) and clustering columns (lo)
func glueLogOffset(hi uint64, low uint16) istructs.Offset {
	return istructs.Offset(hi<<partitionBits | uint64(low))
}

// Returns uint16 as two bytes slice through BigEndian
func uint16bytes(v uint16) []byte {
	b := make([]byte, uint16len)
	binary.BigEndian.PutUint16(b, v)
	return b
}

const uint64len, uint16len = 8, 2

// Returns partition key and clustering columns bytes for specified record id in specified workspace
func recordKey(ws istructs.WSID, id istructs.RecordID) (pkey, ccols []byte) {
	hi, lo := crackRecordID(id)

	pkey = make([]byte, uint16len+uint64len+uint64len)
	binary.BigEndian.PutUint16(pkey, consts.SysView_Records)
	binary.BigEndian.PutUint64(pkey[uint16len:], uint64(ws))
	binary.BigEndian.PutUint64(pkey[uint16len+uint64len:], hi)

	return pkey, uint16bytes(lo)
}

// Returns partition key and clustering columns bytes for specified plog partition and offset
func plogKey(partition istructs.PartitionID, offset istructs.Offset) (pkey, ccols []byte) {
	hi, lo := crackLogOffset(offset)

	pkey = make([]byte, uint16len+uint16len+uint64len)
	binary.BigEndian.PutUint16(pkey, consts.SysView_PLog)
	binary.BigEndian.PutUint16(pkey[uint16len:], uint16(partition))
	binary.BigEndian.PutUint64(pkey[uint16len+uint16len:], hi)

	return pkey, uint16bytes(lo)
}

// Returns partition key and clustering columns bytes for specified wlog workspace and offset
func wlogKey(ws istructs.WSID, offset istructs.Offset) (pkey, ccols []byte) {
	hi, lo := crackLogOffset(offset)

	pkey = make([]byte, uint16len+uint64len+uint64len)
	binary.BigEndian.PutUint16(pkey, consts.SysView_WLog)
	binary.BigEndian.PutUint64(pkey[uint16len:], uint64(ws))
	binary.BigEndian.PutUint64(pkey[uint16len+uint64len:], hi)

	return pkey, uint16bytes(lo)
}

func IBucketsFromIAppStructs(as istructs.IAppStructs) irates.IBuckets {
	// appStructs implementation has method Buckets()
	return as.(interface{ Buckets() irates.IBuckets }).Buckets()
}

func FillElementFromJSON(data map[string]interface{}, def appdef.IDef, b istructs.IElementBuilder) error {
	for fieldName, fieldValue := range data {
		switch fv := fieldValue.(type) {
		case float64:
			b.PutNumber(fieldName, fv)
		case string:
			b.PutChars(fieldName, fv)
		case bool:
			b.PutBool(fieldName, fv)
		case []interface{}:
			// e.g. TestBasicUsage_Dashboard(), "order_item": [<2 elements>]
			containers, ok := def.(appdef.IContainers)
			if !ok {
				return fmt.Errorf("definition %v has no containers", def.QName())
			}
			container := containers.Container(fieldName)
			if container == nil {
				return fmt.Errorf("container with name %s is not found", fieldName)
			}
			for i, val := range fv {
				objContainerElem, ok := val.(map[string]interface{})
				if !ok {
					return fmt.Errorf("element #%d of %s is not an object", i, fieldName)
				}
				containerElemBuilder := b.ElementBuilder(fieldName)
				if err := FillElementFromJSON(objContainerElem, container.Def(), containerElemBuilder); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func NewIObjectBuilder(cfg *AppConfigType, qName appdef.QName) istructs.IObjectBuilder {
	obj := makeObject(cfg, qName)
	return &obj
}

func CheckRefIntegrity(obj istructs.IRowReader, appStructs istructs.IAppStructs, wsid istructs.WSID) (err error) {
	appDef := appStructs.AppDef()
	qName := obj.AsQName(appdef.SystemField_QName)
	def := appDef.Def(qName)
	if fields, ok := def.(appdef.IFields); ok {
		fields.Fields(
			func(f appdef.IField) {
				if f.DataKind() != appdef.DataKind_RecordID || err != nil {
					return
				}
				recID := obj.AsRecordID(f.Name())
				if recID.IsRaw() || recID == istructs.NullRecordID {
					return
				}
				if rec, readErr := appStructs.Records().Get(wsid, true, recID); readErr == nil {
					if rec.QName() == appdef.NullQName {
						err = errors.Join(err,
							fmt.Errorf("%w: record ID %d referenced by %s.%s does not exist", ErrReferentialIntegrityViolation, recID, qName, f.Name()))
					} else {
						if refField, ok := f.(appdef.IRefField); ok {
							if len(refField.Refs()) > 0 && !slices.Contains(refField.Refs(), rec.QName()) {
								err = errors.Join(err,
									fmt.Errorf("%w: record ID %d referenced by %s.%s is of QName %s whereas %v QNames are only allowed", ErrReferentialIntegrityViolation,
										recID, qName, f.Name(), rec.QName(), refField.Refs()))
							}
						}
					}
				} else {
					err = errors.Join(err, readErr)
				}
			})
	}
	return err
}

func NewCmdResultBuilder(appCfg *AppConfigType) istructs.IObjectBuilder {
	obj := makeObject(appCfg, appdef.NewQName(appdef.SysPackage, "TestCmd"))
	return &obj
}
