/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */

package main

import (
	ext "voedger.io/extensions-tinygo"
)

// Handle argument
//
//export exampleCommand
func exampleCommand() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	arg := event.AsValue("arg")

	if event.AsString("qname") == "sys.InvitationAccepted" {
		mail := ext.NewValue(ext.KeyBuilder(ext.StorageSendmail, ext.NullEntity))
		mail.PutString("from", "test@gmail.com")
		mail.PutString("to", arg.AsString("UserEmail"))
		mail.PutString("body", "You are invited")
	}
}

// Handle JSON
//
//export updateSubscriptionProjector
func updateSubscriptionProjector() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))

	if event.AsString("qname") == "air.UpdateSubscription" {
		json := event.AsValue("arg")
		subscr := json.AsValue("subscription")
		customer := json.AsValue("customer")
		mail := ext.NewValue(ext.KeyBuilder(ext.StorageSendmail, ext.NullEntity))
		mail.PutString("from", "test@gmail.com")
		mail.PutString("to", customer.AsString("email"))
		mail.PutString("body", "Your subscription has been updated. New status: "+subscr.AsString("status"))
	}
}

func main() {
}
