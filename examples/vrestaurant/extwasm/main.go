/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Victor Istratenko
 */

package main

import (
	"strconv"

	ext "github.com/heeus/extensions-tinygo"
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
	trExists, tr := ext.QueryValue(kbTr) // replace OK & result in QueryValue
	if trExists {
		return &tr, false
	}
	return nil, true
}

func doUpdateTableStatus(tableNo int32, incAmount int64) {
	kbTS := ext.KeyBuilder(tableStatusQName, strconv.Itoa(int(tableNo)))
	statusExist, statusValue := ext.QueryValue(kbTS)
	if statusExist {
		amount := statusValue.AsInt64("NotPaid") + incAmount
		status := statusValue.AsInt64("Status")
		statusIntent := ext.UpdateValue(kbTS, statusValue)
		statusIntent.PutInt64("NotPaid", amount)
		statusIntent.PutInt32("Amount", int32(incAmount))
		if amount == 0 {
			statusIntent.PutInt32("Status", 0)
		} else if status == 0 {
			statusIntent.PutInt32("Status", 1)
		}
	} else {
		statusIntent := ext.NewValue(kbTS) // Note: why NewValue creates TIntent?
		statusIntent.PutInt32("Status", 1)
		statusIntent.PutInt32("Amount", int32(incAmount))
	}
}

//export updateTableStatus
func updateTableStatus() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	arg := event.AsValue("arg")

	transactionID := arg.AsInt64("transactionID")
	if transactionID == 0 {
		return
	}

	transaction, ok := getTransactionByID(transactionID)
	if !ok {
		return
	}
	tableNo := transaction.AsInt32("Tableno")
	if tableNo == 0 {
		return
	}

	var incAmount int64

	var orderQName = newQName("vrestaurant", "Order")
	var billQName = newQName("vrestaurant", "Bill")
	if (event.AsQName(qn).Entity == orderQName.entity) && (event.AsQName(qn).Pkg == orderQName.pkg) {
		incAmount = getOrderTotal(arg)
	} else if (event.AsQName(qn).Entity == billQName.entity) && (event.AsQName(qn).Pkg == billQName.pkg) {
		incAmount = -getBillTotal(arg)
	}

	if incAmount == 0 {
		return
	}
	doUpdateTableStatus(tableNo, incAmount)

}

//export updateMockTableStatus
func updateMockTableStatus() {
	event := ext.MustGetValue(ext.KeyBuilder(ext.StorageEvent, ext.NullEntity))
	arg := event.AsValue("arg")

	transactionID := arg.AsInt64("transactionID")
	if transactionID == 0 {
		return
	}
	opType := event.AsQName("qname")

	var orderAmount int64
	var billAmount int64
	var orderQName = newQName("vrestaurant", "Order")
	if (opType.Entity == orderQName.entity) && (opType.Pkg == orderQName.pkg) {
		orderAmount = getOrderTotal(arg)
	} else {
		orderAmount = 1560
		billAmount = getBillTotal(arg)
	}

	var total int32
	kbTS := ext.KeyBuilder(tableStatusQName, ext.NullEntity)
	statusIntent := ext.NewValue(kbTS) // Note: why NewValue creates TIntent?
	statusIntent.PutInt32("TableNo", testtablenumber)
	total = int32(orderAmount - billAmount)
	statusIntent.PutInt32("Amount", total)
	if total == 0 {
		statusIntent.PutInt32("Status", 0)
	} else {
		statusIntent.PutInt32("Status", 1)
	}
}

func main() {
}
