/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package main

import (
	ext "github.com/voedger/exttinygo"
)

// BasicUsage test Example Projector.
// Projector calculates the total amount of the ordered items.
//
//export CalcOrderedItems
func CalcOrderedItems() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.Event, ext.NullEntity))
	arg := event.AsValue("ArgumentObject")
	items := arg.AsValue("Items")
	var totalPrice int64
	for i := 0; i < items.Len(); i++ {
		item := items.GetAsValue(i)
		totalPrice += int64(item.AsInt32("Quantity")) * item.AsInt64("SinglePrice")
	}
	key := ext.KeyBuilder(ext.View, "main.OrderedItems")
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

// Handle JSON Example
//
//export updateSubscriptionProjector
func updateSubscriptionProjector() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.Event, ext.NullEntity))

	if event.AsString("qname") == "air.UpdateSubscription" {
		json := event.AsValue("arg")
		subscr := json.AsValue("subscription")
		customer := json.AsValue("customer")
		mail := ext.NewValue(ext.KeyBuilder(ext.SendMail, ext.NullEntity))
		mail.PutString("from", "test@gmail.com")
		mail.PutString("to", customer.AsString("email"))
		mail.PutString("body", "Your subscription has been updated. New status: "+subscr.AsString("status"))
	}
}

//export incrementProjector
func incrementProjector() {
	key := ext.KeyBuilder(ext.View, "pkg.TestView")
	key.PutInt32("pk", 1)
	key.PutInt32("cc", 1)
	value, exists := ext.QueryValue(key)
	if !exists {
		ext.NewValue(key).PutInt32("vv", 1)
	} else {
		ext.UpdateValue(key, value).PutInt32("vv", value.AsInt32("vv")+1)
	}
}

func main() {
}
