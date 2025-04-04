/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
)

func TestResultFieldsOperator_DoSync(t *testing.T) {
	t.Run("Should set result fields", func(t *testing.T) {
		require := require.New(t)

		var (
			appDef  appdef.IAppDef
			rootObj appdef.IObject
		)

		t.Run("Should set result fields", func(t *testing.T) {
			adb := builder.New()

			wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

			addObject := func(n appdef.QName) {
				o := wsb.AddObject(n)
				o.AddField("name", appdef.DataKind_string, false)
			}
			addObject(appdef.NewQName("_", "root"))
			addObject(appdef.NewQName("f", "first_children_1"))
			addObject(appdef.NewQName("f", "deep_children_1"))
			addObject(appdef.NewQName("f", "very_deep_children_1"))
			addObject(appdef.NewQName("s", "first_children_2"))
			addObject(appdef.NewQName("s", "deep_children_1"))
			addObject(appdef.NewQName("s", "very_deep_children_1"))

			objName := appdef.NewQName("test", "root")
			addObject(objName)

			app, err := adb.Build()
			require.NoError(err)

			appDef = app
			rootObj = appdef.Object(app.Type, objName)
		})

		commonFields := []IResultField{resultField{field: "name"}}

		elements := []IElement{
			element{
				path:   path{rootDocument},
				fields: commonFields,
			},
			element{
				path:   path{"first_children_1"},
				fields: commonFields,
			},
			element{
				path:   path{"first_children_1", "deep_children_1"},
				fields: commonFields,
			},
			element{
				path:   path{"first_children_1", "deep_children_1", "very_deep_children_1"},
				fields: commonFields,
			},
			element{
				path:   path{"first_children_2"},
				fields: commonFields,
			},
			element{
				path:   path{"first_children_2", "deep_children_1"},
				fields: commonFields,
			},
			element{
				path:   path{"first_children_2", "deep_children_1", "very_deep_children_1"},
				fields: commonFields,
			},
		}

		work := func() pipeline.IWorkpiece {
			o := &coreutils.TestObject{
				Name:    appdef.NewQName("_", "root"),
				ID_:     istructs.RecordID(1),
				Parent_: istructs.NullRecordID,
				Data: map[string]interface{}{
					"name": "ROOT",
				},
				Containers_: map[string][]*coreutils.TestObject{
					"first_children_1": {
						{
							Name:    appdef.NewQName("f", "first_children_1"),
							ID_:     istructs.RecordID(101),
							Parent_: istructs.RecordID(1),
							Data: map[string]interface{}{
								"name": "FIRST_CHILDREN_1_101",
							},
							Containers_: map[string][]*coreutils.TestObject{
								"deep_children_1": {
									{
										Name:    appdef.NewQName("f", "deep_children_1"),
										ID_:     istructs.RecordID(201),
										Parent_: istructs.RecordID(101),
										Data: map[string]interface{}{
											"name": "DEEP_CHILDREN_1_201",
										},
										Containers_: map[string][]*coreutils.TestObject{
											"very_deep_children_1": {
												{
													Name:    appdef.NewQName("f", "very_deep_children_1"),
													ID_:     istructs.RecordID(301),
													Parent_: istructs.RecordID(201),
													Data: map[string]interface{}{
														"name": "VERY_DEEP_CHILDREN_1_301",
													},
												},
											},
										},
									},
								},
							},
						},
						{
							Name:    appdef.NewQName("f", "first_children_1"),
							ID_:     istructs.RecordID(102),
							Parent_: istructs.RecordID(1),
							Data: map[string]interface{}{
								"name": "FIRST_CHILDREN_1_102",
							},
						},
					},
					"first_children_2": {
						{
							Name:    appdef.NewQName("s", "first_children_2"),
							ID_:     istructs.RecordID(401),
							Parent_: istructs.RecordID(1),
							Data: map[string]interface{}{
								"name": "FIRST_CHILDREN_2_401",
							},
							Containers_: map[string][]*coreutils.TestObject{
								"deep_children_1": {
									{
										Name:    appdef.NewQName("s", "deep_children_1"),
										ID_:     istructs.RecordID(501),
										Parent_: istructs.RecordID(401),
										Data: map[string]interface{}{
											"name": "DEEP_CHILDREN_1_501",
										},
										Containers_: map[string][]*coreutils.TestObject{
											"very_deep_children_1": {
												{
													Name:    appdef.NewQName("s", "very_deep_children_1"),
													ID_:     istructs.RecordID(601),
													Parent_: istructs.RecordID(501),
													Data: map[string]interface{}{
														"name": "VERY_DEEP_CHILDREN_1_601",
													},
												},
												{
													Name:    appdef.NewQName("s", "very_deep_children_1"),
													ID_:     istructs.RecordID(602),
													Parent_: istructs.RecordID(501),
													Data: map[string]interface{}{
														"name": "VERY_DEEP_CHILDREN_1_602",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			return rowsWorkpiece{
				object: o,
				outputRow: &outputRow{
					keyToIdx: map[string]int{
						rootDocument:                       0,
						"first_children_1":                 1,
						"first_children_1/deep_children_1": 2,
						"first_children_1/deep_children_1/very_deep_children_1": 3,
						"first_children_2":                                      4,
						"first_children_2/deep_children_1":                      5,
						"first_children_2/deep_children_1/very_deep_children_1": 6,
					},
					values: make([]interface{}, 7),
				},
			}
		}

		operator := &ResultFieldsOperator{
			elements:   elements,
			rootFields: newFieldsKinds(rootObj),
			fieldsDefs: newFieldsDefs(appDef),
			metrics:    &testMetrics{},
		}

		outWork, err := operator.DoAsync(context.Background(), work())

		require.NoError(err)
		require.Len(outWork.(IWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow), 1)
		require.Equal("ROOT", outWork.(IWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_1").([]IOutputRow), 2)
		require.Equal("FIRST_CHILDREN_1_101", outWork.(IWorkpiece).OutputRow().Value("first_children_1").([]IOutputRow)[0].Value("name"))
		require.Equal("FIRST_CHILDREN_1_102", outWork.(IWorkpiece).OutputRow().Value("first_children_1").([]IOutputRow)[1].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_1/deep_children_1").([]IOutputRow), 1)
		require.Equal("DEEP_CHILDREN_1_201", outWork.(IWorkpiece).OutputRow().Value("first_children_1/deep_children_1").([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_1/deep_children_1/very_deep_children_1").([]IOutputRow), 1)
		require.Equal("VERY_DEEP_CHILDREN_1_301", outWork.(IWorkpiece).OutputRow().Value("first_children_1/deep_children_1/very_deep_children_1").([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_2").([]IOutputRow), 1)
		require.Equal("FIRST_CHILDREN_2_401", outWork.(IWorkpiece).OutputRow().Value("first_children_2").([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1").([]IOutputRow), 1)
		require.Equal("DEEP_CHILDREN_1_501", outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1").([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1/very_deep_children_1").([]IOutputRow), 2)
		require.Equal("VERY_DEEP_CHILDREN_1_601", outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1/very_deep_children_1").([]IOutputRow)[0].Value("name"))
		require.Equal("VERY_DEEP_CHILDREN_1_602", outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1/very_deep_children_1").([]IOutputRow)[1].Value("name"))
	})
	t.Run("Should handle ctx error during row fill with result fields", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		operator := ResultFieldsOperator{
			elements: []IElement{element{path: path{""}, fields: []IResultField{resultField{""}}}},
			metrics:  &testMetrics{},
		}
		work := rowsWorkpiece{
			outputRow: &outputRow{
				keyToIdx: map[string]int{"": 0},
				values:   []interface{}{nil},
			},
		}

		outWork, err := operator.DoAsync(ctx, work)

		require.Equal("context canceled", err.Error())
		require.NotNil(outWork)
	})
	t.Run("Should handle read field value error during row fill with result fields", func(t *testing.T) {
		require := require.New(t)
		work := rowsWorkpiece{
			outputRow: &outputRow{
				keyToIdx: map[string]int{"": 0},
				values:   []interface{}{nil},
			},
		}
		operator := ResultFieldsOperator{
			rootFields: map[string]appdef.DataKind{"": appdef.DataKind_FakeLast},
			elements:   []IElement{element{path: path{""}, fields: []IResultField{resultField{""}}}},
			metrics:    &testMetrics{},
		}

		require.Panics(func() { operator.DoAsync(context.Background(), work) })
	})
	t.Run("Should handle ctx error during row fill with ref fields", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		work := rowsWorkpiece{
			outputRow: &outputRow{
				keyToIdx: map[string]int{"": 0},
				values:   []interface{}{nil},
			},
		}
		operator := ResultFieldsOperator{
			elements: []IElement{element{path: path{""}, refs: []IRefField{refField{"", "", ""}}}},
			metrics:  &testMetrics{},
		}

		outWork, err := operator.DoAsync(ctx, work)

		require.Equal("context canceled", err.Error())
		require.NotNil(outWork)
	})
	t.Run("Should handle ctx error during row fill with result fields from elements", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		work := rowsWorkpiece{
			object: &coreutils.TestObject{Containers_: map[string][]*coreutils.TestObject{
				"container": {&coreutils.TestObject{Data: map[string]interface{}{"": ""}}},
			}},
			outputRow: &outputRow{
				keyToIdx: map[string]int{"": 0},
				values:   []interface{}{nil},
			},
		}
		operator := ResultFieldsOperator{
			fieldsDefs: &fieldsDefs{fields: map[appdef.QName]FieldsKinds{appdef.NullQName: nil}},
			elements:   []IElement{element{path: path{"container"}, fields: []IResultField{resultField{""}}}},
			metrics:    &testMetrics{},
		}

		outWork, err := operator.DoAsync(ctx, work)

		require.Equal("context canceled", err.Error())
		require.NotNil(outWork)
	})
}
