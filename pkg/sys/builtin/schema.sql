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
	)
);
