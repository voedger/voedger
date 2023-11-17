/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package acl0_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/acl0"
)

// MockResourceMatcher is a mock implementation of IResourceMatcher
type MockResourceMatcher struct{}

func (m MockResourceMatcher) Match(r string, rp string) bool {
	return r == rp
}

// MockOperationMatcher is a mock implementation of IOperationMatcher
type MockOperationMatcher struct{}

func (m MockOperationMatcher) Match(o string, op string) bool {
	return o == op
}

func ExampleIACL_HasPermission() {
	// Initialize the ACL builder
	builder := acl0.NewACLBuilder[string, string, string, string, string]()

	// Define roles
	adminRole := "admin"
	userRole := "user"

	// Define resources
	serverResource := "server"
	databaseResource := "database"

	// Define operations
	restartOperation := "restart"
	readOperation := "read"
	writeOperation := "write"

	// Grant permissions
	// Admin can restart the server and read/write the database
	builder.Grant(restartOperation, serverResource, adminRole)
	builder.Grant(readOperation, databaseResource, adminRole)
	builder.Grant(writeOperation, databaseResource, adminRole)

	// Users can only read the database
	builder.Grant(readOperation, databaseResource, userRole)

	// Build the ACL using the mock matchers
	acl := builder.Build(MockResourceMatcher{}, MockOperationMatcher{})

	// Test cases
	fmt.Println("Admin can restart server:", acl.HasPermission(restartOperation, serverResource, adminRole))
	fmt.Println("Admin can read database:", acl.HasPermission(readOperation, databaseResource, adminRole))
	fmt.Println("User can restart server:", acl.HasPermission(restartOperation, serverResource, userRole))
	fmt.Println("User can read database:", acl.HasPermission(readOperation, databaseResource, userRole))
	fmt.Println("User can write database:", acl.HasPermission(writeOperation, databaseResource, userRole))

	// Output:
	// Admin can restart server: true
	// Admin can read database: true
	// User can restart server: false
	// User can read database: true
	// User can write database: false
}
