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
	"math"
	"strings"

	dynobuffers "github.com/untillpro/dynobuffers"
	"github.com/untillpro/voedger/pkg/irates"
	"github.com/untillpro/voedger/pkg/istorage"
	"github.com/untillpro/voedger/pkg/istorageimpl"
	"github.com/untillpro/voedger/pkg/istructs"
	payloads "github.com/untillpro/voedger/pkg/itokens-payloads"
	"github.com/untillpro/voedger/pkg/itokensjwt"
)

var NullAppConfig = newAppConfig(istructs.AppQName_null)

var (
	NullSchema     = newSchema(nil, istructs.NullQName, istructs.SchemaKind_null)
	nullDynoSchema = NullSchema.createDynoBufferScheme()
	nullDynoBuffer = dynobuffers.NewBuffer(nullDynoSchema)
	// not a func -> golang tokensjwt.TimeFunc will be initialized on process init forever
	testTokensFactory    = func() payloads.IAppTokensFactory { return payloads.TestAppTokensFactory(itokensjwt.TestTokensJWT()) }
	simpleStorageProvder = func() istorage.IAppStorageProvider {
		asf := istorage.ProvideMem()
		return istorageimpl.Provide(asf)
	}
)

// copyBytes copies bytes from src
func copyBytes(src []byte) []byte {
	result := make([]byte, len(src))
	copy(result, src)
	return result
}

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

// uncrackLogOffset calculate log offset from two-parts key — partition key (hi) and clustering columns (lo)
func uncrackLogOffset(hi uint64, low uint16) istructs.Offset {
	return istructs.Offset(hi<<partitionBits | uint64(low))
}

const uint64bits, uint16bits = 8, 2

// splitRecordID splits record ID to two-parts key — partition key and clustering columns
func splitRecordID(id istructs.RecordID) (pk, cc []byte) {
	hi, lo := crackRecordID(id)
	pkBuf := make([]byte, uint64bits)
	binary.BigEndian.PutUint64(pkBuf, hi)
	ccBuf := make([]byte, uint16bits)
	binary.BigEndian.PutUint16(ccBuf, lo)
	return pkBuf, ccBuf
}

// splitLogOffset splits offset to two-parts key — partition key and clustering columns
func splitLogOffset(offset istructs.Offset) (pk, cc []byte) {
	hi, lo := crackLogOffset(offset)
	pkBuf := make([]byte, uint64bits)
	binary.BigEndian.PutUint64(pkBuf, hi)
	ccBuf := make([]byte, uint16bits)
	binary.BigEndian.PutUint16(ccBuf, lo)
	return pkBuf, ccBuf
}

// calcLogOffset calculate log offset from two-parts key — partition key and clustering columns
func calcLogOffset(pk, cc []byte) istructs.Offset {
	hi := binary.BigEndian.Uint64(pk)
	low := binary.BigEndian.Uint16(cc)
	return uncrackLogOffset(hi, low)
}

// writeShortString writes short (< 64K) string into a buffer
func writeShortString(buf *bytes.Buffer, str string) {
	const maxLen uint16 = 0xFFFF

	var l uint16
	line := str

	if len(line) < int(maxLen) {
		l = uint16(len(line))
	} else {
		l = maxLen
		line = line[0:maxLen]
	}

	_ = binary.Write(buf, binary.BigEndian, &l)
	buf.WriteString(line)
}

// readShortString reads short (< 64K) string from a buffer
func readShortString(buf *bytes.Buffer) (string, error) {
	var strLen uint16
	if err := binary.Read(buf, binary.BigEndian, &strLen); err != nil {
		return "", fmt.Errorf("error read string length: %w", err)
	}
	if strLen == 0 {
		return "", nil
	}
	if buf.Len() < int(strLen) {
		return "", fmt.Errorf("error read string, expected %d bytes, but only %d bytes is available: %w", strLen, buf.Len(), io.ErrUnexpectedEOF)
	}
	return string(buf.Next(int(strLen))), nil
}

// prefixBytes expands (from left) bytes slice by write specified prefix values. Values must have static size
func prefixBytes(key []byte, prefix ...interface{}) []byte {
	buf := new(bytes.Buffer)
	for _, p := range prefix {
		if err := binary.Write(buf, binary.BigEndian, p); err != nil {
			panic(err)
		}
	}
	if len(key) > 0 {
		_, _ = buf.Write(key)
	}
	return buf.Bytes()
}

// toBytes returns bytes slice constructed from specified values writed from left to right. Values must have static size
func toBytes(prefix ...interface{}) []byte {
	return prefixBytes(nil, prefix...)
}

// fullBytes returns is all bytes is max (0xFF)
func fullBytes(b []byte) bool {
	for _, e := range b {
		if e != math.MaxUint8 {
			return false
		}
	}
	return true
}

// rightMarginCCols returns right margin of half-open range of partially filled clustering columns
func rightMarginCCols(cc []byte) (finishCCols []byte) {
	if fullBytes(cc) {
		return nil
	}

	var incByte func(i int)
	incByte = func(i int) {
		if finishCCols[i] != math.MaxUint8 {
			finishCCols[i] = finishCCols[i] + 1
			return
		}
		finishCCols[i] = 0
		incByte(i - 1)
	}
	finishCCols = make([]byte, len(cc))
	copy(finishCCols, cc)
	incByte(len(cc) - 1)
	return finishCCols
}

// sysField returns is system field name
func sysField(name string) bool {
	return strings.HasPrefix(name, istructs.SystemFieldPrefix) && // fast check
		// then more accuracy
		((name == istructs.SystemField_QName) ||
			(name == istructs.SystemField_ID) ||
			(name == istructs.SystemField_ParentID) ||
			(name == istructs.SystemField_Container) ||
			(name == istructs.SystemField_IsActive))
}

// sysContainer returns is system container name
func sysContainer(name string) bool {
	return strings.HasPrefix(name, istructs.SystemFieldPrefix) && // fast check
		// then more accuracy
		((name == istructs.SystemContainer_ViewValue) ||
			(name == istructs.SystemContainer_ViewPartitionKey) ||
			(name == istructs.SystemContainer_ViewClusteringCols))
}

// validIdent returns is string is valid identifier and error if not
func validIdent(ident string) (ok bool, err error) {
	const (
		char_a rune = 97
		char_A rune = 65
		char_z rune = 122
		char_Z rune = 90
		char_0 rune = 48
		char_9 rune = 57
		char__ rune = 95
	)

	digit := func(r rune) bool {
		return (char_0 <= r) && (r <= char_9)
	}

	letter := func(r rune) bool {
		return ((char_a <= r) && (r <= char_z)) || ((char_A <= r) && (r <= char_Z))
	}

	underScore := func(r rune) bool {
		return r == char__
	}

	if len(ident) < 1 {
		return false, ErrNameMissed
	}

	if len(ident) > MaxIdentLen {
		return false, fmt.Errorf("ident too long: %w", ErrInvalidName)
	}

	for p, c := range ident {
		if !letter(c) && !underScore(c) {
			if (p == 0) || !digit(c) {
				return false, fmt.Errorf("name char «%c» at pos %d is not valid: %w", c, p, ErrInvalidName)
			}
		}
	}

	return true, nil
}

// validQName returns has qName valid package and entity identifiers and error if not
func validQName(qName istructs.QName) (ok bool, err error) {
	if qName == istructs.NullQName {
		return true, nil
	}
	if ok, err = validIdent(qName.Pkg()); !ok {
		return ok, err
	}
	if ok, err = validIdent(qName.Entity()); !ok {
		return ok, err
	}
	return true, nil
}

// used in tests only
func IBucketsFromIAppStructs(as istructs.IAppStructs) irates.IBuckets {
	// appStructs implementation has method Buckets()
	return as.(interface{ Buckets() irates.IBuckets }).Buckets()
}

func FillElementFromJSON(data map[string]interface{}, s istructs.ISchema, b istructs.IElementBuilder, schemas istructs.ISchemas) error {
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
			containerName := fieldName
			var containerQName istructs.QName
			s.Containers(func(cn string, schema istructs.QName) {
				if containerName == cn {
					containerQName = schema
				}
			})
			if containerQName == istructs.NullQName {
				return fmt.Errorf("container with name %s is not found", containerName)
			}
			containerSchema := schemas.Schema(containerQName)
			for i, intf := range fv {
				objContainerElem, ok := intf.(map[string]interface{})
				if !ok {
					return fmt.Errorf("element #%d of %s is not an object", i, fieldName)
				}
				containerElemBuilder := b.ElementBuilder(fieldName)
				if err := FillElementFromJSON(objContainerElem, containerSchema, containerElemBuilder, schemas); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func NewIObjectBuilder(cfg *AppConfigType, qName istructs.QName) istructs.IObjectBuilder {
	obj := newObject(cfg, qName)
	return &obj
}

func CheckRefIntegrity(obj istructs.IRowReader, appStructs istructs.IAppStructs, wsid istructs.WSID) (err error) {
	schemas := appStructs.Schemas()
	schema := schemas.Schema(obj.AsQName(istructs.SystemField_QName))
	schema.ForEachField(func(field istructs.IFieldDescr) {
		if err != nil || field.DataKind() != istructs.DataKind_RecordID {
			return
		}
		recID := obj.AsRecordID(field.Name())
		if recID.IsRaw() || recID == istructs.NullRecordID {
			return
		}
		var rec istructs.IRecord
		rec, err = appStructs.Records().Get(wsid, true, recID)
		if err != nil {
			return
		}
		if rec.QName() == istructs.NullQName {
			err = fmt.Errorf("%w: record ID %d referenced by %s.%s does not exist", ErrReferentialIntegrityViolation, recID,
				obj.AsQName(istructs.SystemField_QName), field.Name())
		}
	})
	return err
}
