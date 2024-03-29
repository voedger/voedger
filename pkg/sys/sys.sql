-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

ABSTRACT TABLE CRecord();

ABSTRACT TABLE WRecord();

ABSTRACT TABLE ORecord();

ABSTRACT TABLE CDoc INHERITS CRecord();

ABSTRACT TABLE ODoc INHERITS ORecord();

ABSTRACT TABLE WDoc INHERITS WRecord();

ABSTRACT TABLE CSingleton INHERITS CDoc();

ABSTRACT TABLE WSingleton INHERITS WDoc();

ALTERABLE WORKSPACE AppWorkspaceWS (
	DESCRIPTOR AppWorkspace ();
);

ABSTRACT WORKSPACE ProfileWS (
	TYPE RefreshPrincipalTokenResult (
		NewPrincipalToken text NOT NULL
	);

	EXTENSION ENGINE BUILTIN (
		QUERY RefreshPrincipalToken RETURNS RefreshPrincipalTokenResult;
	);
);

ALTERABLE WORKSPACE DeviceProfileWS INHERITS ProfileWS (
	DESCRIPTOR DeviceProfile ();
);

TYPE Raw (
	-- must not be bytes because the engine will expect urlBase64-encoded string as the value to put into this field
	Body varchar(65535) NOT NULL
);

EXTENSION ENGINE BUILTIN (
	STORAGE Record(
		GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
		GETBATCH SCOPE(COMMANDS, QUERIES, PROJECTORS),
		INSERT SCOPE(COMMANDS),
		UPDATE SCOPE(COMMANDS)
	) ENTITY RECORD;

	-- used to validate projector state/intents declaration
	STORAGE View(
		GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
		GETBATCH SCOPE(COMMANDS, QUERIES, PROJECTORS),
		READ SCOPE(QUERIES, PROJECTORS),
		INSERT SCOPE(PROJECTORS),
		UPDATE SCOPE(PROJECTORS)
	) ENTITY VIEW;

	STORAGE WLog(
		GET SCOPE(COMMANDS, QUERIES, PROJECTORS),
		READ SCOPE(QUERIES, PROJECTORS)
	);

	STORAGE AppSecret(
		GET SCOPE(COMMANDS, QUERIES, PROJECTORS)
	);

	STORAGE RequestSubject(
		GET SCOPE(COMMANDS, QUERIES)
	);

	STORAGE Http(
		READ SCOPE(QUERIES, PROJECTORS)
	);

	STORAGE SendMail(
		INSERT SCOPE(PROJECTORS)
	);

	STORAGE Result(
		INSERT SCOPE(COMMANDS)
	);

	STORAGE Event(
		GET SCOPE(PROJECTORS)
	);

	STORAGE CommandContext(
		GET SCOPE(COMMANDS)
	);
);
