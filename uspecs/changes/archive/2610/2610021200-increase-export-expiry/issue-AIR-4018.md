# Increase export expiry date

- URL: https://untill.atlassian.net/browse/AIR-4018
- ID: AIR-4018
- State: In Progress
- Author: Michael Saigachenko
- Assignees: Denis Gribanov
- Labels: none
- Linked issues: AIR-3828

## Problem

Due to limitation in the [Temporary BLOB](https://github.com/untillpro/airs-bp3/blob/78d39b1ecdcd36786720671326c0ac4a3b81a939/packages/air/export/impl_handleexport.go#L99) core, the FDM Log exports are only valid 1 day

## Solution

* Support in the core, [as designed](https://internals.voedger.io/server/apiv2/create-tblob/#headers)
* Extend the export [BLOB expiry date](https://github.com/untillpro/airs-bp3/blob/78d39b1ecdcd36786720671326c0ac4a3b81a939/packages/air/export/impl_handleexport.go#L99) to 3 months

    * Synchronize the [export expiration date](https://github.com/untillpro/airs-bp3/blob/78d39b1ecdcd36786720671326c0ac4a3b81a939/packages/air/export/impl_handleexport.go#L107) with the BLOB expiry date

