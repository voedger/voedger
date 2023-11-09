/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package ihttpimpl

type SectionsWriterType interface {
	// Result is false if client cancels the request or receiver is being unregistered
	Write(section interface{}) bool
}
