/*
 * Copyright (c) 2023-present unTill Software Development Group B.V.
 * @author Maxim Geraskin
 */

package acl_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/acl" // Replace with your module path
)

func Example() {
	// Define some roles, operations, and resources
	type Role string
	type Operation string
	type Resource string

	const (
		Admin Role = "Admin"
		User  Role = "User"

		Read      Operation = "Read"
		Shutdown  Operation = "Shutdown"
		Write     Operation = "Write"
		UnknownOp Operation = "UnknownOp"

		File1  Resource = "File1"
		File2  Resource = "File2"
		Server Resource = "Server"
	)

	// Create a new ACL builder
	builder := acl.NewACLBuilder[Role, Operation, Resource]()

	// Grant permissions using the builder
	builder.Grant(Admin, Read, File1)
	builder.Grant(Admin, Write, File1)
	builder.Grant(User, Read, File1)
	builder.Grant(Admin, Shutdown, Server)
	builder.Grant(User, Write, File2)

	// Build the ACL
	myACL := builder.Build()

	// Check some permissions
	fmt.Println("Admin can read File1:", myACL.HasPermission(Admin, Read, File1))           // true
	fmt.Println("User can write File1:", myACL.HasPermission(User, Write, File1))           // false
	fmt.Println("Admin can shutdown Server:", myACL.HasPermission(Admin, Shutdown, Server)) // true
	fmt.Println("User can read Server:", myACL.HasPermission(User, Read, Server))           // false
	fmt.Println("User can write File2:", myACL.HasPermission(User, Write, File2))           // true

	fmt.Println("Admin can UnknownOp File1:", myACL.HasPermission(Admin, UnknownOp, File1)) // false

	// Output:
	// Admin can read File1: true
	// User can write File1: false
	// Admin can shutdown Server: true
	// User can read Server: false
	// User can write File2: true
	// Admin can UnknownOp File1: false
}
