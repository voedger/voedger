/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Victor Istratenko
 */

package main

import (
	"strconv"

	ext "voedger.io/extensions-tinygo"
)

const (
	testtablenumber         = 25
	qn               string = "qname"
	transactionQName        = "vrestaurant.Transaction"
	tableStatusQName        = "vrestaurant.TableStatus"
)

// QName s.e.
type QName struct {
	pkg    string
	entity string
}

func newQName(pkgName, entityName string) QName {
	return QName{pkg: pkgName, entity: entityName}
}

func getOrderTotal(order ext.TValue) int64 {
	orderItem := order.AsValue("OrderItem")
	if orderItem == 0 {
		// try to get value from host
	}
	if orderItem.Length() == 0 { // no items in order
		return 0
	}
	var total int64
	var l = int(orderItem.Length())
	for i := 0; i < l; i++ {
		item := orderItem.GetAsValue(i)
		total = total + int64(item.AsInt64("Quantity")*item.AsInt64("Price"))
	}
	return total
}

func getBillTotal(bill ext.TValue) int64 {
	billPayment := bill.AsValue("BillPayment")
	if billPayment == 0 { // no items in order
		return 0
	}
	if billPayment.Length() == 0 { // no payments in bill
		return 0
	}
	var total int64
	for i := 0; i < int(billPayment.Length()); i++ {
		item := billPayment.GetAsValue(i)
		total = total + item.AsInt64("Amount")
	}
	return total
}

func getTransactionByID(transactionID int64) (transaction *ext.TValue, ok bool) {
	kbTr := ext.KeyBuilder(transactionQName, ext.NullEntity)
	kbTr.PutString("transactionID", strconv.FormatInt(transactionID, 10))
	trExists, trdata := ext.QueryValue(kbTr) // replace OK & result in QueryValue
	if trExists {
		tr := trdata.AsValue("arg")
		return &tr, true
	}
	return nil, false
}

func doUpdateTableStatus(tableNo int64, incAmount int64) {
	kbTS := ext.KeyBuilder(tableStatusQName, ext.NullEntity)
	statusExist, statusValue := ext.QueryValue(kbTS)
	if statusExist {
		amount := statusValue.AsInt64("NotPaid") + int64(incAmount)
		status := statusValue.AsInt32("Status")
		statusIntent := ext.UpdateValue(kbTS, statusValue)
		statusIntent.PutInt64("NotPaid", amount)
		statusIntent.PutInt64("Amount", incAmount)
		if amount == 0 {
			statusIntent.PutInt32("Status", 0)
		} else if status == 0 {
			statusIntent.PutInt32("Status", 1)
		}
	} else {
		statusIntent := ext.NewValue(kbTS) // Note: why NewValue creates TIntent?
		statusIntent.PutInt32("Status", 1)
		statusIntent.PutInt64("Amount", incAmount)
		statusIntent.PutInt64("NotPaid", incAmount)
	}
}

//export updateTableStatus
func updateTableStatus() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	opType := event.AsQName("qname")
	arg := event.AsValue("arg")

	transactionID := arg.AsInt64("transactionID")
	if transactionID == 0 {
		return
	}

	tr, ok := getTransactionByID(transactionID)
	if !ok {
		return
	}
	tableNo := tr.AsInt64("Tableno")
	if tableNo == 0 {
		return
	}

	var incAmount int64
	var orderQName = newQName("vrestaurant", "Order")
	if (opType.Entity == orderQName.entity) && (opType.Pkg == orderQName.pkg) {
		incAmount = getOrderTotal(arg)
	} else {
		incAmount = -getBillTotal(arg)
	}

	if incAmount == 0 {
		return
	}
	doUpdateTableStatus(tableNo, incAmount)
}

func main() {
}
