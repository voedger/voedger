/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
 */

package queryprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func TestResultFieldsOperator_DoSync(t *testing.T) {
	t.Run("Should set result fields", func(t *testing.T) {
		require := require.New(t)
		commonSchema := coreutils.TestSchema{Fields_: map[string]istructs.DataKindType{"name": istructs.DataKind_string}, QName_: istructs.NullQName}
		commonFields := []IResultField{resultField{field: "name"}}
		schemas := coreutils.TestSchemas{Schemas_: map[istructs.QName]istructs.ISchema{
			istructs.NewQName("", "root"):                  commonSchema,
			istructs.NewQName("f", "first-children-1"):     commonSchema,
			istructs.NewQName("f", "deep-children-1"):      commonSchema,
			istructs.NewQName("f", "very-deep-children-1"): commonSchema,
			istructs.NewQName("s", "first-children-2"):     commonSchema,
			istructs.NewQName("s", "deep-children-1"):      commonSchema,
			istructs.NewQName("s", "very-deep-children-1"): commonSchema,
		}}
		elements := []IElement{
			element{
				path:   path{rootDocument},
				fields: commonFields,
			},
			element{
				path:   path{"first-children-1"},
				fields: commonFields,
			},
			element{
				path:   path{"first-children-1", "deep-children-1"},
				fields: commonFields,
			},
			element{
				path:   path{"first-children-1", "deep-children-1", "very-deep-children-1"},
				fields: commonFields,
			},
			element{
				path:   path{"first-children-2"},
				fields: commonFields,
			},
			element{
				path:   path{"first-children-2", "deep-children-1"},
				fields: commonFields,
			},
			element{
				path:   path{"first-children-2", "deep-children-1", "very-deep-children-1"},
				fields: commonFields,
			},
		}

		work := func() pipeline.IWorkpiece {
			o := &coreutils.TestObject{
				Name:    istructs.NewQName("", "root"),
				Id:      istructs.RecordID(1),
				Parent_: istructs.NullRecordID,
				Data: map[string]interface{}{
					"name": "ROOT",
				},
				Containers_: map[string][]*coreutils.TestObject{
					"first-children-1": {
						{
							Name:    istructs.NewQName("f", "first-children-1"),
							Id:      istructs.RecordID(101),
							Parent_: istructs.RecordID(1),
							Data: map[string]interface{}{
								"name": "FIRST-CHILDREN-1-101",
							},
							Containers_: map[string][]*coreutils.TestObject{
								"deep-children-1": {
									{
										Name:    istructs.NewQName("f", "deep-children-1"),
										Id:      istructs.RecordID(201),
										Parent_: istructs.RecordID(101),
										Data: map[string]interface{}{
											"name": "DEEP-CHILDREN-1-201",
										},
										Containers_: map[string][]*coreutils.TestObject{
											"very-deep-children-1": {
												{
													Name:    istructs.NewQName("f", "very-deep-children-1"),
													Id:      istructs.RecordID(301),
													Parent_: istructs.RecordID(201),
													Data: map[string]interface{}{
														"name": "VERY-DEEP-CHILDREN-1-301",
													},
												},
											},
										},
									},
								},
							},
						},
						{
							Name:    istructs.NewQName("f", "first-children-1"),
							Id:      istructs.RecordID(102),
							Parent_: istructs.RecordID(1),
							Data: map[string]interface{}{
								"name": "FIRST-CHILDREN-1-102",
							},
						},
					},
					"first-children-2": {
						{
							Name:    istructs.NewQName("s", "first-children-2"),
							Id:      istructs.RecordID(401),
							Parent_: istructs.RecordID(1),
							Data: map[string]interface{}{
								"name": "FIRST-CHILDREN-2-401",
							},
							Containers_: map[string][]*coreutils.TestObject{
								"deep-children-1": {
									{
										Name:    istructs.NewQName("s", "deep-children-1"),
										Id:      istructs.RecordID(501),
										Parent_: istructs.RecordID(401),
										Data: map[string]interface{}{
											"name": "DEEP-CHILDREN-1-501",
										},
										Containers_: map[string][]*coreutils.TestObject{
											"very-deep-children-1": {
												{
													Name:    istructs.NewQName("s", "very-deep-children-1"),
													Id:      istructs.RecordID(601),
													Parent_: istructs.RecordID(501),
													Data: map[string]interface{}{
														"name": "VERY-DEEP-CHILDREN-1-601",
													},
												},
												{
													Name:    istructs.NewQName("s", "very-deep-children-1"),
													Id:      istructs.RecordID(602),
													Parent_: istructs.RecordID(501),
													Data: map[string]interface{}{
														"name": "VERY-DEEP-CHILDREN-1-602",
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
			return workpiece{
				object: o,
				outputRow: &outputRow{
					keyToIdx: map[string]int{
						rootDocument:                       0,
						"first-children-1":                 1,
						"first-children-1/deep-children-1": 2,
						"first-children-1/deep-children-1/very-deep-children-1": 3,
						"first-children-2":                                      4,
						"first-children-2/deep-children-1":                      5,
						"first-children-2/deep-children-1/very-deep-children-1": 6,
					},
					values: make([]interface{}, 7),
				},
			}
		}

		operator := &ResultFieldsOperator{
			elements:     elements,
			rootSchema:   coreutils.NewSchemaFields(commonSchema),
			schemasCache: newSchemasCache(schemas),
			metrics:      &testMetrics{},
		}

		outWork, err := operator.DoAsync(context.Background(), work())

		require.NoError(err)
		require.Len(outWork.(IWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow), 1)
		require.Equal("ROOT", outWork.(IWorkpiece).OutputRow().Value(rootDocument).([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first-children-1").([]IOutputRow), 2)
		require.Equal("FIRST-CHILDREN-1-101", outWork.(IWorkpiece).OutputRow().Value("first-children-1").([]IOutputRow)[0].Value("name"))
		require.Equal("FIRST-CHILDREN-1-102", outWork.(IWorkpiece).OutputRow().Value("first-children-1").([]IOutputRow)[1].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first-children-1/deep-children-1").([]IOutputRow), 1)
		require.Equal("DEEP-CHILDREN-1-201", outWork.(IWorkpiece).OutputRow().Value("first-children-1/deep-children-1").([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first-children-1/deep-children-1/very-deep-children-1").([]IOutputRow), 1)
		require.Equal("VERY-DEEP-CHILDREN-1-301", outWork.(IWorkpiece).OutputRow().Value("first-children-1/deep-children-1/very-deep-children-1").([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first-children-2").([]IOutputRow), 1)
		require.Equal("FIRST-CHILDREN-2-401", outWork.(IWorkpiece).OutputRow().Value("first-children-2").([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first-children-2/deep-children-1").([]IOutputRow), 1)
		require.Equal("DEEP-CHILDREN-1-501", outWork.(IWorkpiece).OutputRow().Value("first-children-2/deep-children-1").([]IOutputRow)[0].Value("name"))
		require.Len(outWork.(IWorkpiece).OutputRow().Value("first-children-2/deep-children-1/very-deep-children-1").([]IOutputRow), 2)
		require.Equal("VERY-DEEP-CHILDREN-1-601", outWork.(IWorkpiece).OutputRow().Value("first-children-2/deep-children-1/very-deep-children-1").([]IOutputRow)[0].Value("name"))
		require.Equal("VERY-DEEP-CHILDREN-1-602", outWork.(IWorkpiece).OutputRow().Value("first-children-2/deep-children-1/very-deep-children-1").([]IOutputRow)[1].Value("name"))
	})
	t.Run("Should handle ctx error during row fill with result fields", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		operator := ResultFieldsOperator{
			elements: []IElement{element{path: path{""}, fields: []IResultField{resultField{""}}}},
			metrics:  &testMetrics{},
		}
		work := workpiece{
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
		work := workpiece{
			outputRow: &outputRow{
				keyToIdx: map[string]int{"": 0},
				values:   []interface{}{nil},
			},
		}
		operator := ResultFieldsOperator{
			rootSchema: map[string]istructs.DataKindType{"": istructs.DataKind_FakeLast},
			elements:   []IElement{element{path: path{""}, fields: []IResultField{resultField{""}}}},
			metrics:    &testMetrics{},
		}

		require.Panics(func() { operator.DoAsync(context.Background(), work) })
	})
	t.Run("Should handle ctx error during row fill with ref fields", func(t *testing.T) {
		require := require.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		work := workpiece{
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
		work := workpiece{
			object: &coreutils.TestObject{Containers_: map[string][]*coreutils.TestObject{
				"container": {&coreutils.TestObject{Data: map[string]interface{}{"": ""}}},
			}},
			outputRow: &outputRow{
				keyToIdx: map[string]int{"": 0},
				values:   []interface{}{nil},
			},
		}
		operator := ResultFieldsOperator{
			schemasCache: &schemasCache{cache: map[istructs.QName]coreutils.SchemaFields{istructs.NullQName: nil}},
			elements:     []IElement{element{path: path{"container"}, fields: []IResultField{resultField{""}}}},
			metrics:      &testMetrics{},
		}

		outWork, err := operator.DoAsync(ctx, work)

		require.Equal("context canceled", err.Error())
		require.NotNil(outWork)
	})
}
