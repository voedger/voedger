/*
 * Copyright (c) 2024-present unTill Software Development Group B. V. 
 * @author Maxim Geraskin
 */

package schemas

type QName = string

type ID int64

type Entity struct {
	QName QName
}
