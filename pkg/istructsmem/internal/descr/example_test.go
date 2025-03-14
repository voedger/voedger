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
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/appdef/constraints"
	"github.com/voedger/voedger/pkg/appdef/filter"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/descr"
)

func Example() {

	appName := istructs.AppQName_test1_app1

	appDef := func() appdef.IAppDef {
		adb := builder.New()
		adb.AddPackage("test", "test/path")

		wsName, wsDescName := appdef.NewQName("test", "ws"), appdef.NewQName("test", "wsDesc")
		wsb := adb.AddWorkspace(wsName)

		tags := appdef.MustParseQNames("test.dataTag", "test.engineTag", "test.structTag")
		for _, tag := range tags {
			wsb.AddTag(tag, tag.Entity()) // #3363: let use entity name to test tag feature
		}

		numName := appdef.NewQName("test", "number")
		strName := appdef.NewQName("test", "string")

		sysRecords := appdef.NewQName("sys", "records")
		sysViews := appdef.NewQName("sys", "views")

		docName, recName := appdef.NewQName("test", "doc"), appdef.NewQName("test", "rec")

		n := wsb.AddData(numName, appdef.DataKind_int64, appdef.NullQName, constraints.MinIncl(1))
		n.SetComment("natural (positive) integer")
		n.SetTag(tags[0])

		s := wsb.AddData(strName, appdef.DataKind_string, appdef.NullQName)
		s.AddConstraints(constraints.MinLen(1), constraints.MaxLen(100), constraints.Pattern(`^\w+$`, "only word characters allowed"))
		s.SetTag(tags[0])

		doc := wsb.AddCDoc(docName)
		doc.SetSingleton()
		doc.
			AddField("f1", appdef.DataKind_int64, true).
			SetFieldComment("f1", "field comment").
			AddField("f2", appdef.DataKind_string, false, constraints.MinLen(4), constraints.MaxLen(4), constraints.Pattern(`^\w+$`)).
			AddDataField("numField", numName, false).
			AddRefField("mainChild", false, recName)
		doc.AddContainer("rec", recName, 0, 100, "container comment")
		doc.AddUnique(appdef.UniqueQName(docName, "unique1"), []appdef.FieldName{"f1", "f2"})
		doc.SetComment(`comment 1`, `comment 2`)
		doc.SetTag(tags[2])

		rec := wsb.AddCRecord(recName)
		rec.
			AddField("f1", appdef.DataKind_int64, true).
			AddField("f2", appdef.DataKind_string, false).
			AddField("phone", appdef.DataKind_string, true, constraints.MinLen(1), constraints.MaxLen(25)).
			SetFieldVerify("phone", appdef.VerificationKind_Any...)
		rec.
			SetUniqueField("phone").
			AddUnique(appdef.UniqueQName(recName, "uniq1"), []appdef.FieldName{"f1"})
		rec.SetTag(tags[2])

		viewName := appdef.NewQName("test", "view")
		view := wsb.AddView(viewName)
		view.Key().PartKey().
			AddField("pk_1", appdef.DataKind_int64)
		view.Key().ClustCols().
			AddField("cc_1", appdef.DataKind_string, constraints.MaxLen(100))
		view.Value().
			AddDataField("vv_code", strName, true).
			AddRefField("vv_1", true, docName)
		view.SetTag(tags[2])

		objName := appdef.NewQName("test", "obj")
		obj := wsb.AddObject(objName)
		obj.AddField("f1", appdef.DataKind_string, true)
		obj.SetTag(tags[2])

		cmdName := appdef.NewQName("test", "cmd")
		wsb.AddCommand(cmdName).
			SetUnloggedParam(objName).
			SetParam(objName).
			SetEngine(appdef.ExtensionEngineKind_WASM).
			SetTag(tags[1])

		queryName := appdef.NewQName("test", "query")
		wsb.AddQuery(queryName).
			SetParam(objName).
			SetResult(appdef.QNameANY).
			SetTag(tags[1])

		prjRec := wsb.AddProjector(appdef.NewQName("test", "recProjector"))
		prjRec.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Insert, appdef.OperationKind_Update, appdef.OperationKind_Activate, appdef.OperationKind_Deactivate},
			filter.QNames(recName),
			"run projector every time when «test.rec» is changed")
		prjRec.
			SetWantErrors().
			SetEngine(appdef.ExtensionEngineKind_WASM)
		prjRec.States().
			Add(sysRecords, docName, recName).SetComment(sysRecords, "needs to read «test.doc» and «test.rec» from «sys.records» storage")
		prjRec.Intents().
			Add(sysViews, viewName).SetComment(sysViews, "needs to update «test.view» from «sys.views» storage")
		prjRec.SetTag(tags[1])

		prjCmd := wsb.AddProjector(appdef.NewQName("test", "cmdProjector"))
		prjCmd.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_Execute},
			filter.QNames(cmdName),
			"run projector every time when «test.cmd» command is executed")
		prjCmd.SetEngine(appdef.ExtensionEngineKind_WASM)
		prjCmd.SetTag(tags[1])

		prjObj := wsb.AddProjector(appdef.NewQName("test", "objProjector"))
		prjObj.Events().Add(
			[]appdef.OperationKind{appdef.OperationKind_ExecuteWithParam},
			filter.QNames(objName),
			"run projector every time when any command with «test.obj» argument is executed")
		prjObj.SetEngine(appdef.ExtensionEngineKind_WASM)
		prjObj.SetTag(tags[1])

		jobName := appdef.NewQName("test", "job")
		job := wsb.AddJob(jobName)
		job.SetCronSchedule(`@every 1h30m`)
		job.SetEngine(appdef.ExtensionEngineKind_WASM)
		job.States().
			Add(sysViews, viewName).SetComment(sysViews, "needs to read «test.view» from «sys.views» storage")
		job.SetTag(tags[1])

		readerName := appdef.NewQName("test", "reader")
		reader := wsb.AddRole(readerName)
		reader.SetComment("read-only published role")
		reader.SetPublished(true)
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(docName, recName), []appdef.FieldName{"f1", "f2"},
			readerName,
			"allow reader to select some fields from test.doc and test.rec")
		wsb.Grant(
			[]appdef.OperationKind{appdef.OperationKind_Select},
			filter.QNames(viewName), nil,
			readerName,
			"allow reader to select all fields from test.view")
		wsb.GrantAll(filter.QNames(queryName), readerName, "allow reader to execute test.query")

		writerName := appdef.NewQName("test", "writer")
		writer := wsb.AddRole(writerName)
		writer.SetComment("read-write role")
		wsb.GrantAll(filter.QNames(docName, recName), writerName, "allow writer to do anything with test.doc and test.rec")
		wsb.Revoke(
			[]appdef.OperationKind{appdef.OperationKind_Update},
			filter.QNames(docName),
			nil,
			writerName,
			"disable writer to update test.doc")
		wsb.GrantAll(filter.QNames(viewName), writerName, "allow writer to do anything with test.view")
		wsb.GrantAll(filter.AllWSFunctions(wsName), writerName, "allow writer to execute all test functions")

		rateName := appdef.NewQName("test", "rate")
		wsb.AddRate(rateName, 10, time.Minute, []appdef.RateScope{appdef.RateScope_AppPartition}, "rate 10 times per second per partition")

		limitName := appdef.NewQName("test", "limit")
		wsb.AddLimit(limitName, []appdef.OperationKind{appdef.OperationKind_Execute}, appdef.LimitFilterOption_ALL, filter.WSTypes(wsName, appdef.TypeKind_Command), rateName, "limit all commands execution by test.rate")

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
	//           "Tags": {
	//             "test.dataTag": {
	//               "Feature": "dataTag"
	//             },
	//             "test.engineTag": {
	//               "Feature": "engineTag"
	//             },
	//             "test.structTag": {
	//               "Feature": "structTag"
	//             }
	//           },
	//           "DataTypes": {
	//             "test.number": {
	//               "Comment": "natural (positive) integer",
	//               "Tags": [
	//                 "test.dataTag"
	//               ],
	//               "Ancestor": "sys.int64",
	//               "Constraints": {
	//                 "MinIncl": 1
	//               }
	//             },
	//             "test.string": {
	//               "Tags": [
	//                 "test.dataTag"
	//               ],
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
	//               "Tags": [
	//                 "test.structTag"
	//               ],
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
	//               "Tags": [
	//                 "test.structTag"
	//               ],
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
	//               "Tags": [
	//                 "test.structTag"
	//               ],
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
	//               "Tags": [
	//                 "test.structTag"
	//               ],
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
	//                 "Tags": [
	//                   "test.engineTag"
	//                 ],
	//                 "Name": "cmd",
	//                 "Engine": "WASM",
	//                 "Arg": "test.obj",
	//                 "UnloggedArg": "test.obj"
	//               }
	//             },
	//             "Queries": {
	//               "test.query": {
	//                 "Tags": [
	//                   "test.engineTag"
	//                 ],
	//                 "Name": "query",
	//                 "Engine": "BuiltIn",
	//                 "Arg": "test.obj",
	//                 "Result": "sys.ANY"
	//               }
	//             },
	//             "Projectors": {
	//               "test.cmdProjector": {
	//                 "Tags": [
	//                   "test.engineTag"
	//                 ],
	//                 "Name": "cmdProjector",
	//                 "Engine": "WASM",
	//                 "Events": [
	//                   {
	//                     "Ops": [
	//                       "Execute"
	//                     ],
	//                     "Filter": {
	//                       "QNames": [
	//                         "test.cmd"
	//                       ]
	//                     },
	//                     "Comment": "run projector every time when «test.cmd» command is executed"
	//                   }
	//                 ]
	//               },
	//               "test.objProjector": {
	//                 "Tags": [
	//                   "test.engineTag"
	//                 ],
	//                 "Name": "objProjector",
	//                 "Engine": "WASM",
	//                 "Events": [
	//                   {
	//                     "Ops": [
	//                       "ExecuteWithParam"
	//                     ],
	//                     "Filter": {
	//                       "QNames": [
	//                         "test.obj"
	//                       ]
	//                     },
	//                     "Comment": "run projector every time when any command with «test.obj» argument is executed"
	//                   }
	//                 ]
	//               },
	//               "test.recProjector": {
	//                 "Tags": [
	//                   "test.engineTag"
	//                 ],
	//                 "Name": "recProjector",
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
	//                 "Events": [
	//                   {
	//                     "Ops": [
	//                       "Insert",
	//                       "Update",
	//                       "Activate",
	//                       "Deactivate"
	//                     ],
	//                     "Filter": {
	//                       "QNames": [
	//                         "test.rec"
	//                       ]
	//                     },
	//                     "Comment": "run projector every time when «test.rec» is changed"
	//                   }
	//                 ],
	//                 "WantErrors": true
	//               }
	//             },
	//             "Jobs": {
	//               "test.job": {
	//                 "Tags": [
	//                   "test.engineTag"
	//                 ],
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
	//               "Comment": "read-only published role",
	//               "Published": true
	//             },
	//             "test.writer": {
	//               "Comment": "read-write role"
	//             }
	//           },
	//           "ACL": [
	//             {
	//               "Comment": "allow reader to select some fields from test.doc and test.rec",
	//               "Policy": "Allow",
	//               "Ops": [
	//                 "Select"
	//               ],
	//               "Filter": {
	//                 "QNames": [
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
	//               "Filter": {
	//                 "QNames": [
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
	//               "Filter": {
	//                 "QNames": [
	//                   "test.query"
	//                 ]
	//               },
	//               "Principal": "test.reader"
	//             },
	//             {
	//               "Comment": "allow writer to do anything with test.doc and test.rec",
	//               "Policy": "Allow",
	//               "Ops": [
	//                 "Insert",
	//                 "Update",
	//                 "Activate",
	//                 "Deactivate",
	//                 "Select"
	//               ],
	//               "Filter": {
	//                 "QNames": [
	//                   "test.doc",
	//                   "test.rec"
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
	//               "Filter": {
	//                 "QNames": [
	//                   "test.doc"
	//                 ]
	//               },
	//               "Principal": "test.writer"
	//             },
	//             {
	//               "Comment": "allow writer to do anything with test.view",
	//               "Policy": "Allow",
	//               "Ops": [
	//                 "Insert",
	//                 "Update",
	//                 "Select"
	//               ],
	//               "Filter": {
	//                 "QNames": [
	//                   "test.view"
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
	//               "Filter": {
	//                 "Types": [
	//                   "TypeKind_Query",
	//                   "TypeKind_Command"
	//                 ],
	//                 "Workspace": "test.ws"
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
	//               "Ops": [
	//                 "Execute"
	//               ],
	//               "Filter": {
	//                 "Option": "ALL",
	//                 "Types": [
	//                   "TypeKind_Command"
	//                 ],
	//                 "Workspace": "test.ws"
	//               },
	//               "Rate": "test.rate"
	//             }
	//           }
	//         }
	//       }
	//     }
	//   }
	// }
}
