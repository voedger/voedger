/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 * @author Maxim Geraskin
 */

package schemas

import "strconv"

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
