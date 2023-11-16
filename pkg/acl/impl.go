/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package acl

// acl is a struct that implements the IACL interface
type acl[QName comparable] struct {
    // internal fields to store ACL data
    // (e.g., rules, permissions, etc.)
}

// CheckTableAccess checks if access is granted for a table operation
func (a *acl[QName]) CheckTableAccess(op string, fields []string, table QName, roles []QName) bool {
    // Implement the logic to check table access
    // This is just a placeholder implementation
    return true
}

// CheckExecAccess checks if execute access is granted for a resource
func (a *acl[QName]) CheckExecAccess(resource QName, roles []QName) bool {
    // Implement the logic to check execute access
    // This is just a placeholder implementation
    return true
}

// aclBuilder is a struct that implements the IACLBuilder interface
type aclBuilder[QName comparable] struct {
    // internal fields to build the ACL
    // (e.g., rules being constructed, tags, etc.)
}

// TagTable associates a tag with a table
func (b *aclBuilder[QName]) TagTable(tag QName, table QName) IACLBuilder[QName] {
    // Implement the logic to tag a table
    // This is just a placeholder implementation
    return b
}

// GrantOpOnTable grants an operation on a table for a role
func (b *aclBuilder[QName]) GrantOpOnTable(op string, fields []string, table QName, role QName) IACLBuilder[QName] {
    // Implement the logic to grant an operation on a table
    // This is just a placeholder implementation
    return b
}

// GrantOpOnTableByTag grants an operation on tables with a specific tag
func (b *aclBuilder[QName]) GrantOpOnTableByTag(op string, fields []string, tag QName, role QName) IACLBuilder[QName] {
    // Implement the logic to grant an operation on tables by tag
    // This is just a placeholder implementation
    return b
}

// GrantExec grants execute permission on a resource
func (b *aclBuilder[QName]) GrantExec(resource QName, role QName) IACLBuilder[QName] {
    // Implement the logic to grant execute permission
    // This is just a placeholder implementation
    return b
}

// Build finalizes the ACL and returns an instance of IACL
func (b *aclBuilder[QName]) Build() IACL[QName] {
    // Implement the logic to build and return an ACL
    // This is just a placeholder implementation
    return &acl[QName]{}
}
