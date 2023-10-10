-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

ALTER WORKSPACE Workspace (
	TYPE EchoParams (
		Text text NOT NULL
	);

	TYPE EchoResult (
		Res text NOT NULL
	);

	EXTENSION ENGINE BUILTIN (
		QUERY Echo(EchoParams) RETURNS EchoResult;
	);

	VIEW RecordsRegistry (
		IDHi int64 NOT NULL,
		ID ref NOT NULL,
		WLogOffset int64 NOT NULL,
		QName qname NOT NULL,
		PRIMARY KEY ((IDHi), ID)
	) AS RESULT OF sys.RecordsRegistryProjector;
);
