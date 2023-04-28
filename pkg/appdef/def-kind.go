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

//go:generate stringer -type=DefKind -output=def-kind_string.go

const (
	DefKind_null DefKind = iota

	// Глобальный Global configuration, WSID==0 (глобальная номенклатура): UserProfileLocation, SystemConfig
	DefKind_GDoc

	// Кoнфигурационный документ (per workspace articles, prices, clients)
	DefKind_CDoc

	// Operational documents: pbill, orders
	// https://vocable.ru/termin/operacionnyi-dokument.html
	// ОПЕРАЦИОННЫЙ ДОКУМЕНТ счет-фактура, чек, заказ, свидетельствующий о совершении сделки.
	// Might not be edited
	DefKind_ODoc

	// bill
	// Workflow document, extends ODoc
	// Might be edited
	DefKind_WDoc

	// Parts of documents, article_price, pbill_item
	DefKind_GRecord
	DefKind_CRecord
	DefKind_ORecord
	DefKind_WRecord

	// collection (BO)  ((wsid, qname), id), record
	// logins ((wsid0), login) id
	DefKind_ViewRecord
	// No fields with variable length allowed
	DefKind_ViewRecord_PartitionKey
	// Only one variable length field is allowed (must be last field)
	DefKind_ViewRecord_ClusteringColumns
	DefKind_ViewRecord_Value

	// Function params, results, Event.command (this is command function params)
	DefKind_Object
	// Elements of objects
	DefKind_Element

	// Params and Result are DefKind_Object
	DefKind_QueryFunction

	// Params are always ODoc + WDoc
	// Commands have no explicit result
	DefKind_CommandFunction

	DefKind_FakeLast
)

// Is fields allowed.
func (k DefKind) FieldsAllowed() bool {
	return defKindProps[k].fieldsAllowed
}

// Is data kind allowed.
func (k DefKind) DataKindAvailable(d DataKind) bool {
	return defKindProps[k].fieldsAllowed && defKindProps[k].availableFieldKinds[d]
}

// Is specified system field used.
func (k DefKind) HasSystemField(f string) bool {
	return defKindProps[k].fieldsAllowed && defKindProps[k].systemFields[f]
}

// Is containers allowed.
func (k DefKind) ContainersAllowed() bool {
	return defKindProps[k].containersAllowed
}

// Is specified schema kind may be used in child containers.
func (k DefKind) ContainerKindAvailable(s DefKind) bool {
	return defKindProps[k].containersAllowed && defKindProps[k].availableContainerKinds[s]
}

func (k DefKind) MarshalText() ([]byte, error) {
	var s string
	if k < DefKind_FakeLast {
		s = k.String()
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an DefKind in human-readable form, without "DefKind_" prefix,
// suitable for debugging or error messages
func (k DefKind) ToString() string {
	const pref = "DefKind_"
	return strings.TrimPrefix(k.String(), pref)
}
