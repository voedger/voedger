[![codecov](https://codecov.io/gh/voedger/voedger/branch/main/graph/badge.svg?token=1O1pA6zdYs)](https://codecov.io/gh/voedger/voedger/istructsmem/internal/uniques)

# Uniques

Uniques system view.

## Functional design

Application structure provider must call `PrepareApDefUniqueIDs()` to assign IDs for every unique in every structured definition.

`PrepareApDefUniqueIDs(storage istorage.IAppStorage, versions *vers.Versions, qnames *qnames.QNames, appDef appdef.IAppDef)`
- `storage` passed to read previously assigned uniques IDs and to write newly assigned
- `versions` system view  passed to read and write uniques system view version
- `qNames` system view passed to obtain definition QName IDs
- for all `appDef.Defs()` which have uniques() will be assigned IDs.

## Technical design

Prepare particulars:
- IDs for existing uniques reads from storage
- if new uniques exists, then
  + new IDs are generates and assigns to
  + uniques system view storage updated.

Uniques system view storage structure:
- PK => `consts.SysView_UniquesIDs` + view version,
- CC => definition QNameID (hexadecimal string) + field names pipe-separated concatenation, e.g. `"0x000007b5|name|surname|"`
- Value => appdef.UniqueID

See also [unique identification](https://github.com/voedger/voedger/issues/98#:~:text=Unique%20identification)