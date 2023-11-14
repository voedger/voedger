/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author Maxim Geraskin
 */

package appdef

import (
	"strconv"
	"strings"
)

//go:generate stringer -type=TypeKind -output=type-kind_string.go

const (
	TypeKind_null TypeKind = iota

	// Any type.
	//
	// Used as result types kind for functions that has parameter or result of any type.
	TypeKind_Any

	// Simple data types, like string, number, date, etc.
	TypeKind_Data

	// Глобальный Global configuration, WSID==0 (глобальная номенклатура): UserProfileLocation, SystemConfig
	TypeKind_GDoc

	// Конфигурационный документ (per workspace articles, prices, clients)
	TypeKind_CDoc

	// Operational documents: bills, orders
	// https://vocable.ru/termin/operacionnyi-dokument.html
	// ОПЕРАЦИОННЫЙ ДОКУМЕНТ счет-фактура, чек, заказ, свидетельствующий о совершении сделки.
	// Might not be edited
	TypeKind_ODoc

	// bill
	// Workflow document, extends ODoc
	// Might be edited
	TypeKind_WDoc

	// Parts of documents, article_price, bill_item
	TypeKind_GRecord
	TypeKind_CRecord
	TypeKind_ORecord
	TypeKind_WRecord

	// collection (BO)  ((wsid, qname), id), record
	// logins ((wsid0), login) id
	TypeKind_ViewRecord

	// Function params, results, Event.command (this is command function params)
	TypeKind_Object

	// Params and Result are Objects
	TypeKind_Query

	// Params are always Objects.
	// Commands may haven't explicit result
	TypeKind_Command

	TypeKind_Workspace

	TypeKind_FakeLast
)

// Is data kind allowed.
func (k TypeKind) DataKindAvailable(d DataKind) bool {
	return typeKindProps[k].fieldKinds[d]
}

// Is specified system field exists and required.
func (k TypeKind) HasSystemField(f string) (exists, required bool) {
	required, exists = typeKindProps[k].systemFields[f]
	return exists, required
}

// Is specified type kind may be used in child containers.
func (k TypeKind) ContainerKindAvailable(s TypeKind) bool {
	return typeKindProps[k].containerKinds[s]
}

func (k TypeKind) MarshalText() ([]byte, error) {
	var s string
	if k < TypeKind_FakeLast {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an TypeKind in human-readable form, without `TypeKind_` prefix,
// suitable for debugging or error messages
func (k TypeKind) TrimString() string {
	const pref = "TypeKind_"
	return strings.TrimPrefix(k.String(), pref)
}
