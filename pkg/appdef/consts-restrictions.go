/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Maximum identifier length
const MaxIdentLen = 255

// Maximum fields per one structured type
const MaxTypeFieldCount = 65536

// Maximum containers per one structured type
const MaxTypeContainerCount = 65536

// Maximum fields per one unique
const MaxTypeUniqueFieldsCount = 256

// Maximum uniques
const MaxTypeUniqueCount = 100

// Maximum string and bytes data length
const MaxFieldLength = 1024

// Default string and bytes data max length.
//
// This value is used for MaxLen() constraint in system data types `sys.string` and `sys.bytes`.
const DefaultFieldMaxLength = 255
