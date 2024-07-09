/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Alisher Nurmanov
 */

package teststate

type IFullQName interface {
	PkgPath() string
	Entity() string
}

type ICommandRunner interface {
	// methos to fulfill test state
	Record(fQName IFullQName, id int, keyValueList ...any) ICommandRunner
	ArgumentObject(id int, keyValueList ...any) ICommandRunner
	ArgumentObjectRow(path string, id int, keyValueList ...any) ICommandRunner
	// methods to check out the test state
	RequireSingletonInsert(fQName IFullQName, keyValueList ...any) ICommandRunner
	RequireSingletonUpdate(fQName IFullQName, keyValueList ...any) ICommandRunner
	RequireRecordInsert(fQName IFullQName, id int, keyValueList ...any) ICommandRunner
	RequireRecordUpdate(fQName IFullQName, id int, keyValueList ...any) ICommandRunner
	// method to run the test
	Run()
}

type ICommand interface {
	IFullQName
	ArgumentPkgPath() string
	ArgumentEntity() string
	WorkspaceDescriptor() string
}
