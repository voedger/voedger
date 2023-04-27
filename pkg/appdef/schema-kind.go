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

//go:generate stringer -type=SchemaKind -output=schema-kind_string.go

const (
	SchemaKind_null SchemaKind = iota

	// Глобальный Global configuration, WSID==0 (глобальная номенклатура): UserProfileLocation, SystemConfig
	SchemaKind_GDoc

	// Кoнфигурационный документ (per workspace articles, prices, clients)
	SchemaKind_CDoc

	// Operational documents: pbill, orders
	// https://vocable.ru/termin/operacionnyi-dokument.html
	// ОПЕРАЦИОННЫЙ ДОКУМЕНТ счет-фактура, чек, заказ, свидетельствующий о совершении сделки.
	// Might not be edited
	SchemaKind_ODoc

	// bill
	// Workflow document, extends ODoc
	// Might be edited
	SchemaKind_WDoc

	// Parts of documents, article_price, pbill_item
	SchemaKind_GRecord
	SchemaKind_CRecord
	SchemaKind_ORecord
	SchemaKind_WRecord

	// collection (BO)  ((wsid, qname), id), record
	// logins ((wsid0), login) id
	SchemaKind_ViewRecord
	// No fields with variable length allowed
	SchemaKind_ViewRecord_PartitionKey
	// Only one variable length field is allowed (must be last field)
	SchemaKind_ViewRecord_ClusteringColumns
	SchemaKind_ViewRecord_Value

	// Function params, results, Event.command (this is command function params)
	SchemaKind_Object
	// Elements of objects
	SchemaKind_Element

	// Params and Result are SchemaKind_Object
	SchemaKind_QueryFunction

	// Params are always ODoc + WDoc
	// Commands have no explicit result
	SchemaKind_CommandFunction

	SchemaKind_FakeLast
)

// Is fields allowed.
func (k SchemaKind) FieldsAllowed() bool {
	return schemaKindProps[k].fieldsAllowed
}

// Is data kind allowed.
func (k SchemaKind) DataKindAvailable(d DataKind) bool {
	return schemaKindProps[k].fieldsAllowed && schemaKindProps[k].availableFieldKinds[d]
}

// Is specified system field used.
func (k SchemaKind) HasSystemField(f string) bool {
	return schemaKindProps[k].fieldsAllowed && schemaKindProps[k].systemFields[f]
}

// Is containers allowed.
func (k SchemaKind) ContainersAllowed() bool {
	return schemaKindProps[k].containersAllowed
}

// Is specified schema kind may be used in child containers.
func (k SchemaKind) ContainerKindAvailable(s SchemaKind) bool {
	return schemaKindProps[k].containersAllowed && schemaKindProps[k].availableContainerKinds[s]
}

func (k SchemaKind) MarshalText() ([]byte, error) {
	var s string
	if k < SchemaKind_FakeLast {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an SchemaKind in human-readable form, without "SchemaKind_" prefix,
// suitable for debugging or error messages
func (k SchemaKind) ToString() string {
	const pref = "SchemaKind_"
	return strings.TrimPrefix(k.String(), pref)
}
