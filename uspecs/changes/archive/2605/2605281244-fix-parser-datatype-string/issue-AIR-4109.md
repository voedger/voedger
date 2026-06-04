# voedger: fix wrong cases in parser/DataType.String()

- URL: https://untill.atlassian.net/browse/AIR-4109
- ID: AIR-4109
- State: In Progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Description

Why
parser/DataType.String() has wrong result for Float32 (int32) and Float64 (int64) data types

What
return float32 and float64 respectively
