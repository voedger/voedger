# voedger: wait for scylla deploy before tests

- URL: https://untill.atlassian.net/browse/AIR-4407
- ID: AIR-4407
- State: In Progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov

## Why

on Scylla tests on `istorage` modification:

https://github.com/voedger/voedger/actions/runs/28647020113/job/84955965697?pr=4588

```text
=== NAME  TestBasicUsage
    tck.go:49: 
        	Error Trace:	/home/runner/work/voedger/voedger/pkg/istorage/tck.go:49
        	Error:      	Target error should be in err chain:
        	            	expected: "storage does not exist"
        	            	in chain: "can't create session: gocql: unable to create session: unable to discover protocol version: read tcp 127.0.0.1:46672->127.0.0.1:9042: read: connection reset by peer"
        	            		"gocql: unable to create session: unable to discover protocol version: read tcp 127.0.0.1:46672->127.0.0.1:9042: read: connection reset by peer"
        	Test:       	TestBasicUsage
```

## What

wait for scylla deploy in github action
