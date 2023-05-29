SCHEMA sys;

TABLE CDoc(
    ID int64
);

TABLE ODoc(
    ID int64
);

TABLE WDoc(
    ID int64
);

TABLE Singleton INHERITS CDoc (
);