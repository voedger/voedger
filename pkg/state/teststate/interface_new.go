/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Alisher Nurmanov
 */

package teststate

type IFullQName interface {
	PkgPath() string
	Entity() string
}

type IView interface {
	IFullQName
	Keys() []string
}

type ICommandRunner interface {
	// methos to fulfill test state
	Record(fQName IFullQName, id int, keyValueList ...any) ICommandRunner
	SingletonRecord(fQName IFullQName, keyValueList ...any) ICommandRunner
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

type ITestRunner interface {
	CUDRow(fQName IFullQName, id int, keyValueList ...any) ITestRunner
	View(fQName IFullQName, id int, keyValueList ...any) ITestRunner
	Offset(offset int) ITestRunner
	// methos to fulfill test state
	Record(fQName IFullQName, id int, keyValueList ...any) ITestRunner
	SingletonRecord(fQName IFullQName, keyValueList ...any) ITestRunner
	ArgumentObject(id int, keyValueList ...any) ITestRunner
	ArgumentObjectRow(path string, id int, keyValueList ...any) ITestRunner
	// methods to check out the test state
	RequireSingletonInsert(fQName IFullQName, keyValueList ...any) ITestRunner
	RequireSingletonUpdate(fQName IFullQName, keyValueList ...any) ITestRunner
	RequireRecordInsert(fQName IFullQName, id int, keyValueList ...any) ITestRunner
	RequireRecordUpdate(fQName IFullQName, id int, keyValueList ...any) ITestRunner
	RequireViewInsert(fQName IFullQName, keyValueList ...any) ITestRunner
	RequireViewUpdate(fQName IFullQName, id int, keyValueList ...any) ITestRunner
	// method to run the test
	Run()
}

type ICommand interface {
	IFullQName
	ArgumentPkgPath() string
	ArgumentEntity() string
	WorkspaceDescriptor() string
}

type IProjector interface {
	ICommand
}
