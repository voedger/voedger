-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

APPLICATION emptyApp();

TABLE WSKind INHERITS Singleton (
	IntFld int32 NOT NULL,
	StrFld varchar
)