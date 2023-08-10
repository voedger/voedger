-- Copyright (c) 2020-present unTill Pro, Ltd.
-- @author Denis Gribanov

SCHEMA sys;

TABLE UserProfile INHERITS Singleton (
	DisplayName text
);

TABLE DeviceProfile INHERITS Singleton ();

TABLE AppWorkspace INHERITS Singleton ();
