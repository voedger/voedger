-- Copyright (c) 2023-present unTill Pro, Ltd.
-- @author Alisher Nurmanov

IMPORT SCHEMA 'server.com/account/repo/mypkg1';
IMPORT SCHEMA 'server.com/account/repo/mypkg2';
IMPORT SCHEMA 'server.com/account/repo/mypkg3';
IMPORT SCHEMA 'github.com/voedger/voedger/pkg/registry';

APPLICATION test (
	USE mypkg1;
	USE mypkg2;
	USE mypkg3;
	USE registry;
);
