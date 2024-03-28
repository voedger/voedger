/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package main

import (
	ext "github.com/voedger/voedger/pkg/exttinygo"
)

// BasicUsage test Example Command
//
//export NewOrder
func NewOrder() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageCommandContext, ext.NullEntity))
	arg := event.AsValue("ArgumentObject")
	items := arg.AsValue("Items")
	var totalPrice int64
	for i := 0; i < items.Len(); i++ {
		item := items.GetAsValue(i)
		totalPrice += int64(item.AsInt32("Quantity")) * item.AsInt64("SinglePrice")
	}
	if totalPrice <= 0 {
		ext.Panic("negative order amount")
	}
}

// BasicUsage test Example Projector.
// Projector calculates the total amount of the ordered items.
//
//export CalcOrderedItems
func CalcOrderedItems() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	arg := event.AsValue("ArgumentObject")
	items := arg.AsValue("Items")
	var totalPrice int64
	for i := 0; i < items.Len(); i++ {
		item := items.GetAsValue(i)
		totalPrice += int64(item.AsInt32("Quantity")) * item.AsInt64("SinglePrice")
	}
	key := ext.KeyBuilder(ext.StorageView, "github.com/untillpro/airs-bp3/packages/mypkg.OrderedItems")
	key.PutInt32("Year", arg.AsInt32("Year"))
	key.PutInt32("Month", arg.AsInt32("Month"))
	key.PutInt32("Day", arg.AsInt32("Day"))
	value, exists := ext.QueryValue(key)
	if !exists {
		ext.NewValue(key).PutInt64("Amount", totalPrice)
	} else {
		ext.UpdateValue(key, value).PutInt64("Amount", value.AsInt64("Amount")+totalPrice)
	}
}

func main() {
}
