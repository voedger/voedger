/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"encoding/binary"

	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/testingu"

	"github.com/untillpro/dynobuffers"

	"github.com/voedger/voedger/pkg/irates"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istorage/mem"
	"github.com/voedger/voedger/pkg/istorage/provider"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/itokensjwt"
)

var NullAppConfig = newBuiltInAppConfig(istructs.AppQName_null, builder.New())

var (
	nullDynoBuffer = dynobuffers.NewBuffer(dynobuffers.NewScheme())
	// not a func -> golang itokensjwt.TimeFunc will be initialized on process init forever
	testTokensFactory = func() payloads.IAppTokensFactory {
		return payloads.ProvideIAppTokensFactory(itokensjwt.TestTokensJWT())
	}
	simpleStorageProvider = func() istorage.IAppStorageProvider {
		asf := mem.Provide(testingu.MockTime)
		return provider.Provide(asf)
	}
)

// crackID splits ID to two-parts key — partition key (hi) and clustering columns (lo)
func crackID(id uint64) (hi uint64, low uint16) {
	return id >> partitionBits, uint16(id) & lowMask // nolint G115
}

// CrackRecordID splits record ID to two-parts key — partition key (hi) and clustering columns (lo)
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
