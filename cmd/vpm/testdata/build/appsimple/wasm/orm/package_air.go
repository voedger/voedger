/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package orm

import "github.com/voedger/voedger/pkg/exttinygo"

var Package_air = struct {
	Command_Pbill          Command_air_Pbill
	ODoc_ProformaPrinted   ODoc_air_ProformaPrinted
	View_PbillDates        View_air_PbillDates
	WSingleton_NextNumbers WSingleton_air_NextNumbers
}{
	Command_Pbill: Command_air_Pbill{
		Type: Type{qname: "github.com/untillpro/airs-bp3/packages/air.Pbill"},
	},
	ODoc_ProformaPrinted: ODoc_air_ProformaPrinted{
		Type: Type{qname: "github.com/untillpro/airs-bp3/packages/air.ProformaPrinted"},
	},
	View_PbillDates: View_air_PbillDates{
		Type: Type{qname: "github.com/untillpro/airs-bp3/packages/air.PbillDates"},
	},
	WSingleton_NextNumbers: WSingleton_air_NextNumbers{
		Type: Type{qname: "github.com/untillpro/airs-bp3/packages/air.NextNumbers"},
	},
}

/*
COMMAND Pbill(untill.pbill) RETURNS CmdPBillResult;
*/
type Command_air_Pbill struct {
	Type
}

// !!! ArgumentObject result type is defined by the  command statement
//
//	COMMAND Pbill(untill.pbill) RETURNS CmdPBillResult;
func (c Command_air_Pbill) ArgumentObject() Value_ODoc_untill_pbill {

	// !!! return host["StorageCommandContext]["ArgumentObject"]

	kb := exttinygo.KeyBuilder(exttinygo.StorageCommandContext, exttinygo.NullEntity)
	return Value_ODoc_untill_pbill{tv: exttinygo.MustGetValue(kb).AsValue(FieldNameEventArgumentObject)}
}

/*
TYPE CmdPBillResult (

	Number int32 NOT NULL

)
*/
func (c Command_air_Pbill) Result(number int32) {
	result := exttinygo.NewValue(exttinygo.KeyBuilder(exttinygo.StorageResult, exttinygo.NullEntity))
	result.PutInt32("Number", number)
}

/*
TABLE ProformaPrinted INHERITS sys.ODoc (

	Number int32 NOT NULL,
	UserID ref(untill.untill_users) NOT NULL,
	Timestamp int64 NOT NULL,
	BillID ref(untill.bill) NOT NULL

);
*/
type ODoc_air_ProformaPrinted struct {
	Type
}

/*
	TABLE NextNumbers INHERITS sys.CSingleton (
		NextPBillNumber int32
	);
*/

type WSingleton_air_NextNumbers struct {
	Type
}

func (w WSingleton_air_NextNumbers) QueryValue() (value Value_WSingleton_air_NextNumbers, ok bool) {
	kb := exttinygo.KeyBuilder(exttinygo.StorageRecord, w.qname)
	// !!! No key fields since it's a singleton
	tv, ok := exttinygo.QueryValue(kb)
	if !ok {
		return Value_WSingleton_air_NextNumbers{}, false
	}
	kb.PutInt64(FieldNameSysID, tv.AsInt64(FieldNameSysID))
	return Value_WSingleton_air_NextNumbers{tv: tv, kb: kb}, true
}

func (w WSingleton_air_NextNumbers) NewIntent() Intent_WSingleton_air_NextNumbers {
	kb := exttinygo.KeyBuilder(exttinygo.StorageRecord, w.qname)
	// !!! We do not set ID since it's a singleton
	return Intent_WSingleton_air_NextNumbers{intent: exttinygo.NewValue(kb)}
}

type Value_WSingleton_air_NextNumbers struct {
	tv exttinygo.TValue
	kb exttinygo.TKeyBuilder
}

func (v Value_WSingleton_air_NextNumbers) NewIntent() Intent_WSingleton_air_NextNumbers {
	return Intent_WSingleton_air_NextNumbers{intent: exttinygo.NewValue(v.kb)}
}

func (v Value_WSingleton_air_NextNumbers) Get_NextPBillNumber() int32 {
	return v.tv.AsInt32("NextPBillNumber")
}

type Intent_WSingleton_air_NextNumbers struct {
	intent exttinygo.TIntent
}

func (i Intent_WSingleton_air_NextNumbers) Set_NextPBillNumber(value int32) Intent_WSingleton_air_NextNumbers {
	i.intent.PutInt32("NextPBillNumber", value)
	return i
}

/*
VIEW PbillDates (

	Year int32 NOT NULL,
	DayOfYear int32 NOT NULL,
	FirstOffset int64 NOT NULL,
	LastOffset int64 NOT NULL,
	PRIMARY KEY ((Year), DayOfYear)

) AS RESULT OF FillPbillDates;
*/
type View_air_PbillDates struct {
	Type
}

func (v View_air_PbillDates) MustGetValue(year int32, dayOfYear int32) Value_View_untill_PbillDates {
	kb := exttinygo.KeyBuilder(exttinygo.StorageView, v.qname)
	kb.PutInt32("Year", year)
	kb.PutInt32("DayOfYear", dayOfYear)
	return Value_View_untill_PbillDates{tv: exttinygo.MustGetValue(kb), kb: kb}
}

func (v View_air_PbillDates) NewIntent(year int32, dayOfYear int32) Intent_View_untill_PbillDates {
	kb := exttinygo.KeyBuilder(exttinygo.StorageView, v.qname)
	kb.PutInt32("Year", year)
	kb.PutInt32("DayOfYear", dayOfYear)
	return Intent_View_untill_PbillDates{intent: exttinygo.NewValue(kb)}
}

func (v View_air_PbillDates) QueryValue(year int32, dayOfYear int32) (value Value_View_untill_PbillDates, ok bool) {
	kb := exttinygo.KeyBuilder(exttinygo.StorageView, v.qname)
	kb.PutInt32("Year", year)
	kb.PutInt32("DayOfYear", dayOfYear)
	tv, ok := exttinygo.QueryValue(kb)
	if !ok {
		return Value_View_untill_PbillDates{}, false
	}
	return Value_View_untill_PbillDates{tv: tv, kb: kb}, true
}

type Value_View_untill_PbillDates struct {
	tv exttinygo.TValue
	kb exttinygo.TKeyBuilder
}

func (v Value_View_untill_PbillDates) Get_FirstOffset() int32 {
	return v.tv.AsInt32("FirstOffset")
}
func (v Value_View_untill_PbillDates) Get_LastOffset() int32 {
	return v.tv.AsInt32("LastOffset")
}

func (v Value_View_untill_PbillDates) NewIntent() Intent_View_untill_PbillDates {
	return Intent_View_untill_PbillDates{intent: exttinygo.NewValue(v.kb)}
}

type Intent_View_untill_PbillDates struct {
	intent exttinygo.TIntent
}

func (i Intent_View_untill_PbillDates) Set_FirstOffset(value int32) Intent_View_untill_PbillDates {
	i.intent.PutInt32("FirstOffset", value)
	return i
}

func (i Intent_View_untill_PbillDates) Set_LastOffset(value int32) Intent_View_untill_PbillDates {
	i.intent.PutInt32("LastOffset", value)
	return i
}

func (i Intent_View_untill_PbillDates) Intent() exttinygo.TIntent {
	return i.intent
}
