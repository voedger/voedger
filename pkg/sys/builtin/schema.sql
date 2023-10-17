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

		-- Login is randomly taken name because it is required to specify something in the sql. Actually the projector will start on an any document.
		SYNC PROJECTOR RecordsRegistryProjector ON (Login) INTENTS(View(RecordsRegistry));
	);

	VIEW RecordsRegistry (
		IDHi int64 NOT NULL,
		ID ref NOT NULL,
		WLogOffset int64 NOT NULL,
		QName qname NOT NULL,
		PRIMARY KEY ((IDHi), ID)
	) AS RESULT OF sys.RecordsRegistryProjector;
);
