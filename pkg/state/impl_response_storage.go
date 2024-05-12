/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package state

import (
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type cmdResponseStorage struct {
}

type responseKeyBuilder struct {
	*keyBuilder
}

func newResponseKeyBuilder() *responseKeyBuilder {
	return &responseKeyBuilder{
		&keyBuilder{
			storage: Response,
		},
	}
}
func (*responseKeyBuilder) Equals(src istructs.IKeyBuilder) bool {
	_, ok := src.(*responseKeyBuilder)
	return ok
}

type responseValueBuilder struct {
	*baseValueBuilder
	statusCode   int32
	errorMessage string
}

func (b *responseValueBuilder) PutInt32(name string, value int32) {
	switch name {
	case Field_StatusCode:
		b.statusCode = value
	default:
		panic(errUndefined(name))
	}
}

func (b *responseValueBuilder) PutString(name string, value string) {
	switch name {
	case Field_ErrorMessage:
		b.errorMessage = value
	default:
		panic(errUndefined(name))
	}
}

func (b *responseValueBuilder) BuildValue() istructs.IStateValue {
	return &responsesValue{
		baseStateValue: &baseStateValue{},
		statusCode:     b.statusCode,
		errorMessage:   b.errorMessage,
	}
}

type responsesValue struct {
	*baseStateValue
	statusCode   int32
	errorMessage string
}

func (v *responsesValue) AsInt32(name string) int32 {
	switch name {
	case Field_StatusCode:
		return v.statusCode
	default:
		panic(errUndefined(name))
	}
}

func (v *responsesValue) AsString(name string) string {
	switch name {
	case Field_ErrorMessage:
		return v.errorMessage
	default:
		panic(errUndefined(name))
	}
}

func (s *cmdResponseStorage) NewKeyBuilder(_ appdef.QName, _ istructs.IStateKeyBuilder) istructs.IStateKeyBuilder {
	return newResponseKeyBuilder()
}

func (s *cmdResponseStorage) Validate([]ApplyBatchItem) (err error) {
	panic("not applicable")
}

func (s *cmdResponseStorage) ApplyBatch([]ApplyBatchItem) (err error) {
	panic("not applicable")
}

func (s *cmdResponseStorage) ProvideValueBuilder(istructs.IStateKeyBuilder, istructs.IStateValueBuilder) (istructs.IStateValueBuilder, error) {
	return &responseValueBuilder{
		&baseValueBuilder{},
		http.StatusOK,
		"",
	}, nil
}
