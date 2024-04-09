-- Copyright (c) 2024-present unTill Software Development Group B.V.
-- @author Denis Gribanov

WORKSPACE ClusterWS (
	DESCRIPTOR Cluster();

	TYPE VSqlQueryParams (
		Query text NOT NULL
	);

	TYPE VSqlQueryResult (
		Result text NOT NULL
	)

	EXTENSION ENGINE BUILTIN (
		QUERY VSqlQuery(VSqlQueryParams) RETURNS VSqlQueryResult;
	);
);
