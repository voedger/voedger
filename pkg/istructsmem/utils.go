/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"encoding/binary"
	"fmt"

	"github.com/untillpro/dynobuffers"

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

// сrackID splits ID to two-parts key — partition key (hi) and clustering columns (lo)
func crackID(id uint64) (hi uint64, low uint16) {
	return uint64(id >> partitionBits), uint16(id) & lowMask
}

// СrackRecordID splits record ID to two-parts key — partition key (hi) and clustering columns (lo)
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

// TODO: @nnv: eliminate the parameter «t appdef.IType».
//   - To obtain object name builder should use b.String() interface
//   - If some complex field (with value type is []interface{}) has ChildBuilder with NullQName, then return error
func FillObjectFromJSON(data map[string]interface{}, t appdef.IType, b istructs.IObjectBuilder) error {
	for fieldName, fieldValue := range data {
		switch fv := fieldValue.(type) {
		case float64:
			b.PutNumber(fieldName, fv)
		case string:
			b.PutChars(fieldName, fv)
		case bool:
			b.PutBool(fieldName, fv)
		case []interface{}:
			// e.g. "order_item": [<2 children>]
			containers, ok := t.(appdef.IContainers)
			if !ok {
				return fmt.Errorf("type %v has no containers", t.QName())
			}
			container := containers.Container(fieldName)
			if container == nil {
				return fmt.Errorf("container with name %s is not found", fieldName)
			}
			for i, val := range fv {
				childData, ok := val.(map[string]interface{})
				if !ok {
					return fmt.Errorf("child #%d of %s is not an object", i, fieldName)
				}
				childBuilder := b.ChildBuilder(fieldName)
				if err := FillObjectFromJSON(childData, container.Type(), childBuilder); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// TODO: @nnv: deprecate this function!
//
//	Add new IAppStructs.ObjectBuilder(appdef.QName).
//	AppConfigType should be used internally
func NewIObjectBuilder(cfg *AppConfigType, qName appdef.QName) istructs.IObjectBuilder {
	return newObject(cfg, qName, nil)
}
