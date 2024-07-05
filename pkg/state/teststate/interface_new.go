package teststate

type IFullQName interface {
	PkgPath() string
	Entity() string
}

type ICommandRunner interface {
	Record(fQName IFullQName, id int, keyValueList ...any) ICommandRunner
	ArgumentObject(id int, keyValueList ...any) ICommandRunner
	ArgumentObjectRow(path string, id int, keyValueList ...any) ICommandRunner
	Run() ICommandRequire
}

type ICommand interface {
	IFullQName
	ArgumentPkgPath() string
	ArgumentEntity() string
	WorkspaceDescriptor() string
}

type ICommandRequire interface {
	SingletonInsert(fQName IFullQName, keyValueList ...any)
	SingletonUpdate(fQName IFullQName, keyValueList ...any)
	RecordInsert(fQName IFullQName, id int, keyValueList ...any)
	RecordUpdate(fQName IFullQName, id int, keyValueList ...any)
}
