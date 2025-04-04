/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/sys"
)

func TestEnrichmentOperator_DoSync(t *testing.T) {

	t.Run("Should set reference fields", func(t *testing.T) {
		require := require.New(t)

		appDef := func() appdef.IAppDef {
			adb := builder.New()
			wsb := adb.AddWorkspace(appdef.NewQName("test", "workspace"))

			addObject := func(n appdef.QName) {
				wsb.AddObject(n).
					AddField("id_lower_case_name", appdef.DataKind_RecordID, false)
			}

			addObject(appdef.NewQName("_", "root"))
			addObject(appdef.NewQName("f", "first_children_1"))
			addObject(appdef.NewQName("f", "deep_children_1"))
			addObject(appdef.NewQName("f", "very_deep_children_1"))
			addObject(appdef.NewQName("s", "first_children_2"))
			addObject(appdef.NewQName("s", "deep_children_1"))
			addObject(appdef.NewQName("s", "very_deep_children_1"))

			wsb.AddObject(qNameXLowerCase).
				AddField("name", appdef.DataKind_string, false)

			app, err := adb.Build()
			require.NoError(err)

			return app
		}()

		commonFields := []IRefField{refField{field: "id_lower_case_name", ref: "name", key: "id_lower_case_name/name"}}

		elements := []IElement{
			element{
				path: path{rootDocument},
				refs: commonFields,
			},
			element{
				path: path{"first_children_1"},
				refs: commonFields,
			},
			element{
				path: path{"first_children_1", "deep_children_1"},
				refs: commonFields,
			},
			element{
				path: path{"first_children_1", "deep_children_1", "very_deep_children_1"},
				refs: commonFields,
			},
			element{
				path: path{"first_children_2"},
				refs: commonFields,
			},
			element{
				path: path{"first_children_2", "deep_children_1"},
				refs: commonFields,
			},
			element{
				path: path{"first_children_2", "deep_children_1", "very_deep_children_1"},
				refs: commonFields,
			},
		}
		row := func(idLowerCaseName int) IOutputRow {
			return &outputRow{
				keyToIdx: map[string]int{
					"id_lower_case_name/name": 0,
				},
				values: []interface{}{
					istructs.RecordID(idLowerCaseName),
				},
			}
		}
		record := func(name string) istructs.IStateValue {
			r := &mockRecord{}
			r.
				On("AsString", "name").Return(name).
				On("QName").Return(qNameXLowerCase)
			sv := &mockStateValue{}
			sv.On("AsRecord", "").Return(r)
			return sv
		}
		work := func() pipeline.IWorkpiece {
			o := &coreutils.TestObject{
				Name:    appdef.NewQName("", "root"),
				ID_:     istructs.RecordID(1),
				Parent_: istructs.NullRecordID,
				Data: map[string]interface{}{
					"id_lower_case_name": istructs.RecordID(2001),
					"name":               "ROOT",
				},
				Containers_: map[string][]*coreutils.TestObject{
					"first_children_1": {
						{
							Name:    appdef.NewQName("f", "first_children_1"),
							ID_:     istructs.RecordID(101),
							Parent_: istructs.RecordID(1),
							Data: map[string]interface{}{
								"id_lower_case_name": istructs.RecordID(200101),
								"name":               "FIRST_CHILDREN_1_101",
							},
							Containers_: map[string][]*coreutils.TestObject{
								"deep_children_1": {
									{
										Name:    appdef.NewQName("f", "deep_children_1"),
										ID_:     istructs.RecordID(201),
										Parent_: istructs.RecordID(101),
										Data: map[string]interface{}{
											"id_lower_case_name": istructs.RecordID(200201),
											"name":               "DEEP_CHILDREN_1_201",
										},
										Containers_: map[string][]*coreutils.TestObject{
											"very_deep_children_1": {
												{
													Name:    appdef.NewQName("f", "very_deep_children_1"),
													ID_:     istructs.RecordID(301),
													Parent_: istructs.RecordID(201),
													Data: map[string]interface{}{
														"id_lower_case_name": istructs.RecordID(200301),
														"name":               "VERY_DEEP_CHILDREN_1_301",
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
								"id_lower_case_name": istructs.RecordID(200102),
								"name":               "FIRST_CHILDREN_1_102",
							},
						},
					},
					"first_children_2": {
						{
							Name:    appdef.NewQName("s", "first_children_2"),
							ID_:     istructs.RecordID(401),
							Parent_: istructs.RecordID(1),
							Data: map[string]interface{}{
								"id_lower_case_name": istructs.RecordID(200401),
								"name":               "FIRST_CHILDREN_2_401",
							},
							Containers_: map[string][]*coreutils.TestObject{
								"deep_children_1": {
									{
										Name:    appdef.NewQName("s", "deep_children_1"),
										ID_:     istructs.RecordID(501),
										Parent_: istructs.RecordID(401),
										Data: map[string]interface{}{
											"id_lower_case_name": istructs.RecordID(200501),
											"name":               "DEEP_CHILDREN_1_501",
										},
										Containers_: map[string][]*coreutils.TestObject{
											"very_deep_children_1": {
												{
													Name:    appdef.NewQName("s", "very_deep_children_1"),
													ID_:     istructs.RecordID(601),
													Parent_: istructs.RecordID(501),
													Data: map[string]interface{}{
														"id_lower_case_name": istructs.RecordID(200601),
														"name":               "VERY_DEEP_CHILDREN_1_601",
													},
												},
												{
													Name:    appdef.NewQName("s", "very_deep_children_1"),
													ID_:     istructs.RecordID(602),
													Parent_: istructs.RecordID(501),
													Data: map[string]interface{}{
														"id_lower_case_name": istructs.RecordID(200602),
														"name":               "VERY_DEEP_CHILDREN_1_602",
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
					values: []interface{}{
						[]IOutputRow{row(2001)},
						[]IOutputRow{row(200101), row(200102)},
						[]IOutputRow{row(200201)},
						[]IOutputRow{row(200301)},
						[]IOutputRow{row(200401)},
						[]IOutputRow{row(200501)},
						[]IOutputRow{row(200601), row(200602)},
					},
				},
				enrichedRootFieldsKinds: make(map[string]appdef.DataKind),
			}
		}
		skb := &mockStateKeyBuilder{}
		skb.On("PutRecordID", mock.Anything, mock.Anything)
		s := &mockState{}
		s.
			On("KeyBuilder", sys.Storage_Record, appdef.NullQName).Return(skb).
			On("MustExist", mock.Anything).Return(record("root")).Once().
			On("MustExist", mock.Anything).Return(record("first_children_1_101")).Once().
			On("MustExist", mock.Anything).Return(record("first_children_1_102")).Once().
			On("MustExist", mock.Anything).Return(record("deep_children_1_201")).Once().
			On("MustExist", mock.Anything).Return(record("very_deep_children_1_301")).Once().
			On("MustExist", mock.Anything).Return(record("first_children_2_401")).Once().
			On("MustExist", mock.Anything).Return(record("deep_children_1_501")).Once().
			On("MustExist", mock.Anything).Return(record("very_deep_children_1_601")).Once().
			On("MustExist", mock.Anything).Return(record("very_deep_children_1_602")).Once()
		op := &EnrichmentOperator{
			state:      s,
			elements:   elements,
			fieldsDefs: newFieldsDefs(appDef),
			metrics:    &testMetrics{},
		}

		outWork, err := op.DoAsync(context.Background(), work())

		require.NoError(err)
		require.Len(outWork.(IWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow), 1)
		require.Equal("root", outWork.(IWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Value("id_lower_case_name/name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_1").([]IOutputRow), 2)
		require.Equal("first_children_1_101", outWork.(IWorkpiece).OutputRow().Value("first_children_1").([]IOutputRow)[0].Value("id_lower_case_name/name"))
		require.Equal("first_children_1_102", outWork.(IWorkpiece).OutputRow().Value("first_children_1").([]IOutputRow)[1].Value("id_lower_case_name/name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_1/deep_children_1").([]IOutputRow), 1)
		require.Equal("deep_children_1_201", outWork.(IWorkpiece).OutputRow().Value("first_children_1/deep_children_1").([]IOutputRow)[0].Value("id_lower_case_name/name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_1/deep_children_1/very_deep_children_1").([]IOutputRow), 1)
		require.Equal("very_deep_children_1_301", outWork.(IWorkpiece).OutputRow().Value("first_children_1/deep_children_1/very_deep_children_1").([]IOutputRow)[0].Value("id_lower_case_name/name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_2").([]IOutputRow), 1)
		require.Equal("first_children_2_401", outWork.(IWorkpiece).OutputRow().Value("first_children_2").([]IOutputRow)[0].Value("id_lower_case_name/name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1").([]IOutputRow), 1)
		require.Equal("deep_children_1_501", outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1").([]IOutputRow)[0].Value("id_lower_case_name/name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1/very_deep_children_1").([]IOutputRow), 2)
		require.Equal("very_deep_children_1_601", outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1/very_deep_children_1").([]IOutputRow)[0].Value("id_lower_case_name/name"))
		require.Equal("very_deep_children_1_602", outWork.(IWorkpiece).OutputRow().Value("first_children_2/deep_children_1/very_deep_children_1").([]IOutputRow)[1].Value("id_lower_case_name/name"))
	})
	t.Run("Should handle ctx error during row fill with ref fields", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		work := rowsWorkpiece{
			outputRow: &outputRow{
				keyToIdx: map[string]int{rootDocument: 0},
				values: []interface{}{
					[]IOutputRow{&outputRow{}},
				},
			},
		}
		op := EnrichmentOperator{
			elements: []IElement{element{path: path{""}, refs: []IRefField{refField{"", "", ""}}}},
			metrics:  &testMetrics{},
		}

		outWork, err := op.DoAsync(ctx, work)

		require.Equal("context canceled", err.Error())
		require.Equal(work, outWork)
	})
}
