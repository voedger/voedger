/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package istructsmem

import (
	"errors"
	"sync"

	"github.com/voedger/voedger/pkg/appdef"
)

type fieldValidator struct {
	v   *validator
	err error

	closureValidRequired func(f appdef.IField)
	row                  *rowType

	closureValidPK func(f appdef.IField)
	closureValidCC func(f appdef.IField)
	key            *keyType
	pkDef          appdef.QName
	ccDef          appdef.QName
}

func (fv *fieldValidator) validRequired(f appdef.IField) {
	if f.Required() {
		if !fv.row.HasValue(f.Name()) {
			fv.err = errors.Join(fv.err,
				validateErrorf(ECode_EmptyData, "%s misses field «%s» required by definition «%v»: %w", fv.v.entName(fv.row), f.Name(), fv.v.def.QName(), ErrNameNotFound))
		}
	}
}

func (fv *fieldValidator) validPK(f appdef.IField) {
	if !fv.key.partRow.HasValue(f.Name()) {
		fv.err = errors.Join(fv.err,
			validateErrorf(ECode_EmptyData, "view «%v» partition key «%v» field «%s» is empty: %w", fv.key.viewName, fv.pkDef, f.Name(), ErrFieldIsEmpty))
	}
}

func (fv *fieldValidator) validCC(f appdef.IField) {
	if !fv.key.ccolsRow.HasValue(f.Name()) {
		fv.err = errors.Join(fv.err,
			validateErrorf(ECode_EmptyData, "view «%v» clustering columns «%v» field «%s» is empty: %w", fv.key.viewName, fv.ccDef, f.Name(), ErrFieldIsEmpty))
	}
}

var fieldValidatorPool = sync.Pool{
	New: func() interface{} {
		fv := &fieldValidator{}
		fv.closureValidRequired = fv.validRequired
		fv.closureValidPK = fv.validPK
		fv.closureValidCC = fv.validCC
		return fv
	},
}

func fieldValidatorPoolGet_validKey(key *keyType) *fieldValidator {
	fv := fieldValidatorPool.Get().(*fieldValidator)
	fv.err = nil
	fv.key = key
	fv.pkDef = key.pkDef()
	fv.ccDef = key.ccDef()
	return fv
}

func fieldValidatorPoolPut_validKey(fv *fieldValidator) {
	fv.err = nil
	fv.key = nil
	fv.pkDef = appdef.NullQName
	fv.ccDef = appdef.NullQName
	fieldValidatorPool.Put(fv)
}

func fieldValidatorPoolGet_validRequired(v *validator, row *rowType) *fieldValidator {
	fv := fieldValidatorPool.Get().(*fieldValidator)
	fv.v = v
	fv.row = row
	fv.err = nil
	return fv
}

func fieldValidatorPoolPut_validRequired(fv *fieldValidator) {
	fv.v = nil
	fv.row = nil
	fv.err = nil
	fieldValidatorPool.Put(fv)
}
