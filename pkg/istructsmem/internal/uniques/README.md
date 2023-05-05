[![codecov](https://codecov.io/gh/voedger/voedger/branch/main/graph/badge.svg?token=1O1pA6zdYs)](https://codecov.io/gh/voedger/voedger/istructsmem/internal/uniques)

# Uniques

Uniques system view.

## Functional design

Uniques system view members:
- ID(appdef.IUnique) (appdef.UniqueID, error)
- Prepare(appdef.AppDef)

## Technical design

Prepare particulars:
- IDs for existing uniques reads from storage
- if new uniques exists, then
  + new IDs are generates and assigns to
  + system view Uniques updates to storage.

Storage:
- PK => `consts.SysView_UniquesIDs` + view version,
- CC => definition QNameID + field names pipe-separated concatenation, e.g. `"0x000007b5|name|surname|"`
- Value => appdef.UniqueID

See also [unique identification](https://github.com/voedger/voedger/issues/98#:~:text=Unique%20identification)