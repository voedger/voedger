-- note: this schema is for tests only. Voedger sys package uses copy of this schema
SCHEMA test_sys;
TABLE CDoc();
TABLE ODoc();
TABLE WDoc();
TABLE Singleton INHERITS CDoc();
TABLE CRecord();
TABLE WRecord();
TABLE ORecord();
