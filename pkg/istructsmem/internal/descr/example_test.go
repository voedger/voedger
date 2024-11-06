/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/descr"
)

func Example() {

	appName := istructs.AppQName_test1_app1

	appDef := func() appdef.IAppDef {
		adb := appdef.New()
		adb.AddPackage("test", "test/path")

		wsName, wsDescName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "wsDesc")
		wsb := adb.AddWorkspace(wsName)

		numName := appdef.NewQName("test", "number")
		strName := appdef.NewQName("test", "string")

		sysRecords := appdef.NewQName("sys", "records")
		sysViews := appdef.NewQName("sys", "views")

		docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

		n := wsb.AddData(numName, appdef.DataKind_int64, appdef.NullQName, appdef.MinIncl(1))
		n.SetComment("natural (positive) integer")

		s := wsb.AddData(strName, appdef.DataKind_string, appdef.NullQName)
		s.AddConstraints(appdef.MinLen(1), appdef.MaxLen(100), appdef.Pattern(`^\w+$`, "only word characters allowed"))

		doc := wsb.AddCDoc(docName)
		doc.SetSingleton()
		doc.
			AddField("f1", appdef.DataKind_int64, true).
			SetFieldComment("f1", "field comment").
			AddField("f2", appdef.DataKind_string, false, appdef.MinLen(4), appdef.MaxLen(4), appdef.Pattern(`^\w+$`)).
			AddDataField("numField", numName, false).
			AddRefField("mainChild", false, recName)
		doc.AddContainer("rec", recName, 0, 100, "container comment")
		doc.AddUnique(appdef.UniqueQName(docName, "unique1"), []appdef.FieldName{"f1", "f2"})
		doc.SetComment(`comment 1`, `comment 2`)

		rec := wsb.AddCRecord(recName)
		rec.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false).
			AddField("phone", appdef.DataKind_string, true, appdef.MinLen(1), appdef.MaxLen(25)).
			SetFieldVerify("phone", appdef.VerificationKind_Any...)
		rec.
			SetUniqueField("phone").
			AddUnique(appdef.UniqueQName(recName, "uniq1"), []appdef.FieldName{"f1"})

		viewName := appdef.NewQName("test", "view")
		view := wsb.AddView(viewName)
		view.Key().PartKey().
			AddField("pk_1", appdef.DataKind_int64)
		view.Key().ClustCols().
			AddField("cc_1", appdef.DataKind_string, appdef.MaxLen(100))
		view.Value().
			AddDataField("vv_code", strName, true).
			AddRefField("vv_1", true, docName)

		objName := appdef.NewQName("test", "obj")
		obj := wsb.AddObject(objName)
		obj.AddField("f1", appdef.DataKind_string, true)

		cmdName := appdef.NewQName("test", "cmd")
		wsb.AddCommand(cmdName).
			SetUnloggedParam(objName).
			SetParam(objName).
			SetEngine(appdef.ExtensionEngineKind_WASM)

		queryName := appdef.NewQName("test", "query")
		wsb.AddQuery(queryName).
			SetParam(objName).
			SetResult(appdef.QNameANY)

		prjName := appdef.NewQName("test", "projector")
		prj := wsb.AddProjector(prjName)
		prj.
			SetWantErrors().
			SetEngine(appdef.ExtensionEngineKind_WASM)
		prj.Events().
			Add(recName, appdef.ProjectorEventKind_AnyChanges...).SetComment(recName, "run projector every time when «test.rec» is changed").
			Add(cmdName).SetComment(cmdName, "run projector every time when «test.cmd» command is executed").
			Add(objName).SetComment(objName, "run projector every time when any command with «test.obj» argument is executed")
		prj.States().
			Add(sysRecords, docName, recName).SetComment(sysRecords, "needs to read «test.doc» and «test.rec» from «sys.records» storage")
		prj.Intents().
			Add(sysViews, viewName).SetComment(sysViews, "needs to update «test.view» from «sys.views» storage")

		jobName := appdef.NewQName("test", "job")
		job := wsb.AddJob(jobName)
		job.SetCronSchedule(`@every 1h30m`)
		job.SetEngine(appdef.ExtensionEngineKind_WASM)
		job.States().
			Add(sysViews, viewName).SetComment(sysViews, "needs to read «test.view» from «sys.views» storage")

		readerName := appdef.NewQName("test", "reader")
		reader := wsb.AddRole(readerName)
		reader.SetComment("read-only role")
		reader.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			[]appdef.QName{docName, recName}, []appdef.FieldName{"f1", "f2"},
			"allow reader to select some fields from test.doc and test.rec")
		reader.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			[]appdef.QName{viewName}, nil,
			"allow reader to select all fields from test.view")
		reader.GrantAll([]appdef.QName{queryName}, "allow reader to execute test.query")

		writerName := appdef.NewQName("test", "writer")
		writer := wsb.AddRole(writerName)
		writer.SetComment("read-write role")
		writer.GrantAll([]appdef.QName{docName, recName, viewName}, "allow writer to do anything with test.doc, test.rec and test.view")
		writer.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			[]appdef.QName{docName},
			nil,
			"disable writer to update test.doc")
		writer.GrantAll([]appdef.QName{cmdName, queryName}, "allow writer to execute all test functions")

		rateName := appdef.NewQName("test", "rate")
		wsb.AddRate(rateName, 10, time.Minute, []appdef.RateScope{appdef.RateScope_AppPartition}, "rate 10 times per second per partition")

		limitName := appdef.NewQName("test", "limit")
		wsb.AddLimit(limitName, []appdef.QName{appdef.QNameAnyCommand}, rateName, "limit all commands execution by test.rate")

		wsb.AddCDoc(wsDescName).SetSingleton()
		wsb.SetDescriptor(wsDescName)

		app, err := adb.Build()
		if err != nil {
			panic(err)
		}

		return app
	}()

	app := descr.Provide(appName, appDef)

	json, err := json.MarshalIndent(app, "", "  ")

	fmt.Println("error:", err)
	fmt.Println(string(json))

	//os.WriteFile("C://temp//provide_test.json", json, 0644)

	// Output:
	// error: <nil>
	// {
	//   "Name": "test1/app1",
	//   "Packages": {
	//     "test": {
	//       "Path": "test/path",
	//       "Workspaces": {
	//         "test.ws": {
	//           "Descriptor": "test.wsDesc",
	//           "DataTypes": {
	//             "test.number": {
	//               "Comment": "natural (positive) integer",
	//               "Ancestor": "sys.int64",
	//               "Constraints": {
	//                 "MinIncl": 1
	//               }
	//             },
	//             "test.string": {
	//               "Ancestor": "sys.string",
	//               "Constraints": {
	//                 "MaxLen": 100,
	//                 "MinLen": 1,
	//                 "Pattern": "^\\w+$"
	//               }
	//             }
	//           },
	//           "Structures": {
	//             "test.doc": {
	//               "Comment": "comment 1\ncomment 2",
	//               "Kind": "CDoc",
	//               "Fields": [
	//                 {
	//                   "Name": "sys.QName",
	//                   "Data": "sys.QName",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "sys.ID",
	//                   "Data": "sys.RecordID",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "sys.IsActive",
	//                   "Data": "sys.bool"
	//                 },
	//                 {
	//                   "Comment": "field comment",
	//                   "Name": "f1",
	//                   "Data": "sys.int64",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "f2",
	//                   "DataType": {
	//                     "Ancestor": "sys.string",
	//                     "Constraints": {
	//                       "MaxLen": 4,
	//                       "MinLen": 4,
	//                       "Pattern": "^\\w+$"
	//                     }
	//                   }
	//                 },
	//                 {
	//                   "Name": "numField",
	//                   "Data": "test.number"
	//                 },
	//                 {
	//                   "Name": "mainChild",
	//                   "Data": "sys.RecordID",
	//                   "Refs": [
	//                     "test.rec"
	//                   ]
	//                 }
	//               ],
	//               "Containers": [
	//                 {
	//                   "Comment": "container comment",
	//                   "Name": "rec",
	//                   "Type": "test.rec",
	//                   "MinOccurs": 0,
	//                   "MaxOccurs": 100
	//                 }
	//               ],
	//               "Uniques": {
	//                 "test.doc$uniques$unique1": {
	//                   "Fields": [
	//                     "f1",
	//                     "f2"
	//                   ]
	//                 }
	//               },
	//               "Singleton": true
	//             },
	//             "test.obj": {
	//               "Kind": "Object",
	//               "Fields": [
	//                 {
	//                   "Name": "sys.QName",
	//                   "Data": "sys.QName",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "sys.Container",
	//                   "Data": "sys.string"
	//                 },
	//                 {
	//                   "Name": "f1",
	//                   "Data": "sys.string",
	//                   "Required": true
	//                 }
	//               ]
	//             },
	//             "test.rec": {
	//               "Kind": "CRecord",
	//               "Fields": [
	//                 {
	//                   "Name": "sys.QName",
	//                   "Data": "sys.QName",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "sys.ID",
	//                   "Data": "sys.RecordID",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "sys.ParentID",
	//                   "Data": "sys.RecordID",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "sys.Container",
	//                   "Data": "sys.string",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "sys.IsActive",
	//                   "Data": "sys.bool"
	//                 },
	//                 {
	//                   "Name": "f1",
	//                   "Data": "sys.int64",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "f2",
	//                   "Data": "sys.string"
	//                 },
	//                 {
	//                   "Name": "phone",
	//                   "DataType": {
	//                     "Ancestor": "sys.string",
	//                     "Constraints": {
	//                       "MaxLen": 25,
	//                       "MinLen": 1
	//                     }
	//                   },
	//                   "Required": true,
	//                   "Verifiable": true
	//                 }
	//               ],
	//               "Uniques": {
	//                 "test.rec$uniques$uniq1": {
	//                   "Fields": [
	//                     "f1"
	//                   ]
	//                 }
	//               },
	//               "UniqueField": "phone"
	//             },
	//             "test.wsDesc": {
	//               "Kind": "CDoc",
	//               "Fields": [
	//                 {
	//                   "Name": "sys.QName",
	//                   "Data": "sys.QName",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "sys.ID",
	//                   "Data": "sys.RecordID",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "sys.IsActive",
	//                   "Data": "sys.bool"
	//                 }
	//               ],
	//               "Singleton": true
	//             }
	//           },
	//           "Views": {
	//             "test.view": {
	//               "Key": {
	//                 "Partition": [
	//                   {
	//                     "Name": "pk_1",
	//                     "Data": "sys.int64",
	//                     "Required": true
	//                   }
	//                 ],
	//                 "ClustCols": [
	//                   {
	//                     "Name": "cc_1",
	//                     "DataType": {
	//                       "Ancestor": "sys.string",
	//                       "Constraints": {
	//                         "MaxLen": 100
	//                       }
	//                     }
	//                   }
	//                 ]
	//               },
	//               "Value": [
	//                 {
	//                   "Name": "sys.QName",
	//                   "Data": "sys.QName",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "vv_code",
	//                   "Data": "test.string",
	//                   "Required": true
	//                 },
	//                 {
	//                   "Name": "vv_1",
	//                   "Data": "sys.RecordID",
	//                   "Required": true,
	//                   "Refs": [
	//                     "test.doc"
	//                   ]
	//                 }
	//               ]
	//             }
	//           },
	//           "Extensions": {
	//             "Commands": {
	//               "test.cmd": {
	//                 "Name": "cmd",
	//                 "Engine": "WASM",
	//                 "Arg": "test.obj",
	//                 "UnloggedArg": "test.obj"
	//               }
	//             },
	//             "Queries": {
	//               "test.query": {
	//                 "Name": "query",
	//                 "Engine": "BuiltIn",
	//                 "Arg": "test.obj",
	//                 "Result": "sys.ANY"
	//               }
	//             },
	//             "Projectors": {
	//               "test.projector": {
	//                 "Name": "projector",
	//                 "Engine": "WASM",
	//                 "States": {
	//                   "sys.records": [
	//                     "test.doc",
	//                     "test.rec"
	//                   ]
	//                 },
	//                 "Intents": {
	//                   "sys.views": [
	//                     "test.view"
	//                   ]
	//                 },
	//                 "Events": {
	//                   "test.cmd": {
	//                     "Comment": "run projector every time when «test.cmd» command is executed",
	//                     "Kind": [
	//                       "Execute"
	//                     ]
	//                   },
	//                   "test.obj": {
	//                     "Comment": "run projector every time when any command with «test.obj» argument is executed",
	//                     "Kind": [
	//                       "ExecuteWithParam"
	//                     ]
	//                   },
	//                   "test.rec": {
	//                     "Comment": "run projector every time when «test.rec» is changed",
	//                     "Kind": [
	//                       "Insert",
	//                       "Update",
	//                       "Activate",
	//                       "Deactivate"
	//                     ]
	//                   }
	//                 },
	//                 "WantErrors": true
	//               }
	//             },
	//             "Jobs": {
	//               "test.job": {
	//                 "Name": "job",
	//                 "Engine": "WASM",
	//                 "States": {
	//                   "sys.views": [
	//                     "test.view"
	//                   ]
	//                 },
	//                 "CronSchedule": "@every 1h30m"
	//               }
	//             }
	//           },
	//           "Roles": {
	//             "test.reader": {
	//               "Comment": "read-only role",
	//               "ACL": [
	//                 {
	//                   "Comment": "allow reader to select some fields from test.doc and test.rec",
	//                   "Policy": "Allow",
	//                   "Ops": [
	//                     "Select"
	//                   ],
	//                   "Resources": {
	//                     "On": [
	//                       "test.doc",
	//                       "test.rec"
	//                     ],
	//                     "Fields": [
	//                       "f1",
	//                       "f2"
	//                     ]
	//                   }
	//                 },
	//                 {
	//                   "Comment": "allow reader to select all fields from test.view",
	//                   "Policy": "Allow",
	//                   "Ops": [
	//                     "Select"
	//                   ],
	//                   "Resources": {
	//                     "On": [
	//                       "test.view"
	//                     ]
	//                   }
	//                 },
	//                 {
	//                   "Comment": "allow reader to execute test.query",
	//                   "Policy": "Allow",
	//                   "Ops": [
	//                     "Execute"
	//                   ],
	//                   "Resources": {
	//                     "On": [
	//                       "test.query"
	//                     ]
	//                   }
	//                 }
	//               ]
	//             },
	//             "test.writer": {
	//               "Comment": "read-write role",
	//               "ACL": [
	//                 {
	//                   "Comment": "allow writer to do anything with test.doc, test.rec and test.view",
	//                   "Policy": "Allow",
	//                   "Ops": [
	//                     "Insert",
	//                     "Update",
	//                     "Select"
	//                   ],
	//                   "Resources": {
	//                     "On": [
	//                       "test.doc",
	//                       "test.rec",
	//                       "test.view"
	//                     ]
	//                   }
	//                 },
	//                 {
	//                   "Comment": "disable writer to update test.doc",
	//                   "Policy": "Deny",
	//                   "Ops": [
	//                     "Update"
	//                   ],
	//                   "Resources": {
	//                     "On": [
	//                       "test.doc"
	//                     ]
	//                   }
	//                 },
	//                 {
	//                   "Comment": "allow writer to execute all test functions",
	//                   "Policy": "Allow",
	//                   "Ops": [
	//                     "Execute"
	//                   ],
	//                   "Resources": {
	//                     "On": [
	//                       "test.cmd",
	//                       "test.query"
	//                     ]
	//                   }
	//                 }
	//               ]
	//             }
	//           },
	//           "ACL": [
	//             {
	//               "Comment": "allow reader to select some fields from test.doc and test.rec",
	//               "Policy": "Allow",
	//               "Ops": [
	//                 "Select"
	//               ],
	//               "Resources": {
	//                 "On": [
	//                   "test.doc",
	//                   "test.rec"
	//                 ],
	//                 "Fields": [
	//                   "f1",
	//                   "f2"
	//                 ]
	//               },
	//               "Principal": "test.reader"
	//             },
	//             {
	//               "Comment": "allow reader to select all fields from test.view",
	//               "Policy": "Allow",
	//               "Ops": [
	//                 "Select"
	//               ],
	//               "Resources": {
	//                 "On": [
	//                   "test.view"
	//                 ]
	//               },
	//               "Principal": "test.reader"
	//             },
	//             {
	//               "Comment": "allow reader to execute test.query",
	//               "Policy": "Allow",
	//               "Ops": [
	//                 "Execute"
	//               ],
	//               "Resources": {
	//                 "On": [
	//                   "test.query"
	//                 ]
	//               },
	//               "Principal": "test.reader"
	//             },
	//             {
	//               "Comment": "allow writer to do anything with test.doc, test.rec and test.view",
	//               "Policy": "Allow",
	//               "Ops": [
	//                 "Insert",
	//                 "Update",
	//                 "Select"
	//               ],
	//               "Resources": {
	//                 "On": [
	//                   "test.doc",
	//                   "test.rec",
	//                   "test.view"
	//                 ]
	//               },
	//               "Principal": "test.writer"
	//             },
	//             {
	//               "Comment": "disable writer to update test.doc",
	//               "Policy": "Deny",
	//               "Ops": [
	//                 "Update"
	//               ],
	//               "Resources": {
	//                 "On": [
	//                   "test.doc"
	//                 ]
	//               },
	//               "Principal": "test.writer"
	//             },
	//             {
	//               "Comment": "allow writer to execute all test functions",
	//               "Policy": "Allow",
	//               "Ops": [
	//                 "Execute"
	//               ],
	//               "Resources": {
	//                 "On": [
	//                   "test.cmd",
	//                   "test.query"
	//                 ]
	//               },
	//               "Principal": "test.writer"
	//             }
	//           ],
	//           "Rates": {
	//             "test.rate": {
	//               "Comment": "rate 10 times per second per partition",
	//               "Count": 10,
	//               "Period": 60000000000,
	//               "Scopes": [
	//                 "AppPartition"
	//               ]
	//             }
	//           },
	//           "Limits": {
	//             "test.limit": {
	//               "Comment": "limit all commands execution by test.rate",
	//               "On": [
	//                 "sys.AnyCommand"
	//               ],
	//               "Rate": "test.rate"
	//             }
	//           }
	//         }
	//       }
	//     }
	//   }
	// }
}
