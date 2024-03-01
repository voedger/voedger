-- Copyright (c) 2024-present unTill Software Development Group B.V.
-- @author Denis Gribanov

APPLICATION app2();

ALTERABLE WORKSPACE test_wsWS (
	TYPE GreeterParams (
		Text varchar
	);

	TYPE GreeterResult (
		Res varchar
	);

	EXTENSION ENGINE BUILTIN (
		QUERY Greeter(GreeterParams) RETURNS GreeterResult;
	);
);
