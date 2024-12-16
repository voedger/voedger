/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Enumeration of operation kinds.
//
// Used for ACL and for Limits.
type OperationKind uint8

//go:generate stringer -type=OperationKind -output=stringer_operationkind.go

const (
	OperationKind_null OperationKind = iota

	// # Insert records or view records.
	// 	- Operation applicable on records, view records.
	// 	- Fields are not applicable.
	OperationKind_Insert

	// # Update records or view records.
	// 	- Operation applicable on records, view records.
	// 	- Fields are applicable and specify fields of records or view records that can be updated.
	OperationKind_Update

	// # Select records or view records.
	// 	- Operation applicable on records, view records.
	// 	- Fields are applicable and specify fields of records or view records that can be selected.
	OperationKind_Select

	// # Execute functions.
	// 	- Operation applicable on commands, queries.
	// 	- Fields are not applicable.
	OperationKind_Execute

	// # ACL Only. Inherit ACL rules from other roles.
	// 	- Operation applicable on roles only.
	// 	- Fields are not applicable.
	OperationKind_Inherits

	OperationKind_count
)
