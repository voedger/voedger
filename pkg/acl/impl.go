/*
 * Copyright (c) 2023-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package acl

type ACL[QName comparable] struct {
    // Define the necessary fields to store your ACL rules
    // For example, a map to store table access rules and exec access rules
    tableAccessRules map[QName]map[string]map[string][]QName // table -> op -> fields -> roles
    execAccessRules  map[QName][]QName                       // resource -> roles
}

func NewACL[QName comparable]() *ACL[QName] {
    return &ACL[QName]{
        tableAccessRules: make(map[QName]map[string]map[string][]QName),
        execAccessRules:  make(map[QName][]QName),
    }
}

func (acl *ACL[QName]) CheckTableAccess(op string, fields []string, table QName, roles []QName) bool {
    // Implement the logic to check table access
    // This would typically involve looking up the table, op, and fields in the acl.tableAccessRules
    // and then checking if any of the provided roles are allowed
}

func (acl *ACL[QName]) CheckExecAccess(resource QName, roles []QName) bool {
    // Implement the logic to check exec access
    // This would involve looking up the resource in the acl.execAccessRules
    // and then checking if any of the provided roles are allowed
}

// aclBuilder is a mock implementation of the IACLBuilder interface
type aclBuilder[QName comparable] struct {
	acl *ACL[QName]
}

// newACLBuilder creates a new instance of MockACLBuilder
func newACLBuilder[QName comparable]() *aclBuilder[QName] {
	return &aclBuilder[QName]{}
}

// Role implements the Role method of the IACLBuilder interface
func (m *aclBuilder[QName]) Role(name QName) IACLBuilder[QName] {
	// Mock implementation
	return m
}

// GrantOpOnTable implements the GrantOpOnTable method of the IACLBuilder interface
func (m *aclBuilder[QName]) GrantOpOnTable(op string, fields []string, table QName, role QName) IACLBuilder[QName] {
	// Mock implementation
	return m
}

// GrantOpOnTableByTag implements the GrantOpOnTableByTag method of the IACLBuilder interface
func (m *aclBuilder[QName]) GrantOpOnTableByTag(op string, fields []string, tag QName, role QName) IACLBuilder[QName] {
	// Mock implementation
	return m
}

// GrantExec implements the GrantExec method of the IACLBuilder interface
func (m *aclBuilder[QName]) GrantExec(resource QName, role QName) IACLBuilder[QName] {
	// Mock implementation
	return m
}

// Build implements the Build method of the IACLBuilder interface
func (m *aclBuilder[QName]) Build() IACL[QName] {
	// Mock implementation
	// Return a mock IACL instance or nil, depending on your testing needs
	return nil
}

// Ensure MockACLBuilder implements IACLBuilder
var _ IACLBuilder[string] = (*aclBuilder[string])(nil)
