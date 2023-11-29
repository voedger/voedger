IMPORT SCHEMA 'server.com/account/repo/mypkgerr';
IMPORT SCHEMA 'github.com/voedger/voedger/pkg/registry';

APPLICATION test (
	USE mypkgerr;
	USE registry;
);
