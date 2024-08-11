/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

type cmdResponseStorage struct {
}

type responseKeyBuilder struct {
	baseKeyBuilder
}

func (b *responseKeyBuilder) Storage() appdef.QName {
	return sys.Storage_Response
}

func (b *responseKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*responseKeyBuilder)
	return ok
}

type responseValueBuilder struct {
	baseValueBuilder
	statusCode   int32
	errorMessage string
}

func (b *responseValueBuilder) PutInt32(name string, value int32) {
	switch name {
	case sys.Storage_Response_Field_StatusCode:
		b.statusCode = value
	default:
		b.baseValueBuilder.PutInt32(name, value)
	}
}

func (b *responseValueBuilder) PutString(name string, value string) {
	switch name {
	case sys.Storage_Response_Field_ErrorMessage:
		b.errorMessage = value
	default:
		b.baseValueBuilder.PutString(name, value)
	}
}

func (b *responseValueBuilder) BuildValue() istructs.IStateValue {
	return &responsesValue{
		statusCode:   b.statusCode,
		errorMessage: b.errorMessage,
	}
}

type responsesValue struct {
	baseStateValue
	statusCode   int32
	errorMessage string
}

func (v *responsesValue) AsInt32(name string) int32 {
	switch name {
	case sys.Storage_Response_Field_StatusCode:
		return v.statusCode
	default:
		return v.baseStateValue.AsInt32(name)
	}
}

func (v *responsesValue) AsString(name string) string {
	switch name {
	case sys.Storage_Response_Field_ErrorMessage:
		return v.errorMessage
	default:
		return v.baseStateValue.AsString(name)
	}
}

func (s *cmdResponseStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return &responseKeyBuilder{}
}

func (s *cmdResponseStorage) Validate([]ApplyBatchItem) (err error) {
	return nil
}

func (s *cmdResponseStorage) ApplyBatch([]ApplyBatchItem) (err error) {
	return nil
}

func (s *cmdResponseStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return &responseValueBuilder{
		baseValueBuilder{},
		http.StatusOK,
		"",
	}, nil
}
