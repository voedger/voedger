/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"embed"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

//go:embed sql_example_app/pmain/*.vsql
var fsMain embed.FS

//go:embed sql_example_app/airsbp/*.vsql
var fsAir embed.FS

//go:embed sql_example_app/untill/*.vsql
var fsUntill embed.FS

//go:embed sql_example_syspkg/*.vsql
var sfs embed.FS

//go:embed sql_example_app/vrestaurant/*.vsql
var fsvRestaurant embed.FS

//_go:embed example_app/expectedParsed.schema
// var expectedParsedExampledSchemaStr string

func getSysPackageAST() *PackageSchemaAST {
	pkgSys, err := ParsePackageDir(appdef.SysPackage, sfs, "sql_example_syspkg")
	if err != nil {
		panic(err)
	}
	return pkgSys
}

func Test_BasicUsage(t *testing.T) {

	require := require.New(t)
	mainPkgAST, err := ParsePackageDir("github.com/untillpro/main", fsMain, "sql_example_app/pmain")
	require.NoError(err)

	airPkgAST, err := ParsePackageDir("github.com/untillpro/airsbp", fsAir, "sql_example_app/airsbp")
	require.NoError(err)

	untillPkgAST, err := ParsePackageDir("github.com/untillpro/untill", fsUntill, "sql_example_app/untill")
	require.NoError(err)

	appSchema, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		mainPkgAST,
		airPkgAST,
		untillPkgAST,
	})
	require.NoError(err)

	builder := appdef.New()
	err = BuildAppDefs(appSchema, builder)
	require.NoError(err)

	app, err := builder.Build()
	require.NoError(err)

	// table
	cdoc := appdef.CDoc(app.Type, appdef.NewQName("main", "TablePlan"))
	require.NotNil(cdoc)
	require.Equal(appdef.TypeKind_CDoc, cdoc.Kind())
	require.Equal(appdef.DataKind_int32, cdoc.Field("FState").DataKind())
	require.Equal("Backoffice Table", cdoc.Comment())

	// TODO: sf := cdoc.Field("CheckedField").(appdef.IStringField)
	// TODO: require.Equal(uint16(8), sf.Restricts().MaxLen())
	// TODO: require.NotNil(sf.Restricts().Pattern())

	// container of the table
	container := cdoc.Container("TableItems")
	require.Equal("TableItems", container.Name())
	require.Equal(appdef.NewQName("main", "TablePlanItem"), container.QName())
	require.Equal(appdef.Occurs(0), container.MinOccurs())
	require.Equal(appdef.Occurs(maxNestedTableContainerOccurrences), container.MaxOccurs())
	require.Equal(appdef.TypeKind_CRecord, container.Type().Kind())
	require.Equal(2+5 /* +5 system fields*/, container.Type().(appdef.IFields).FieldCount())
	require.Equal(appdef.DataKind_int32, container.Type().(appdef.IFields).Field("TableNo").DataKind())
	require.Equal(appdef.DataKind_int32, container.Type().(appdef.IFields).Field("Chairs").DataKind())

	// constraint
	uniques := cdoc.Uniques()
	require.Len(uniques, 2)

	t.Run("first unique, automatically named", func(t *testing.T) {
		u := uniques[appdef.MustParseQName("main.TablePlan$uniques$01")]
		require.NotNil(u)
		cnt := 0
		for _, f := range u.Fields() {
			cnt++
			switch n := f.Name(); n {
			case "FState":
				require.Equal(appdef.DataKind_int32, f.DataKind())
			case "Name":
				require.Equal(appdef.DataKind_string, f.DataKind())
			default:
				require.Fail("unexpected field name", n)
			}
		}
		require.Equal(2, cnt)
	})

	t.Run("second unique, named by user", func(t *testing.T) {
		u := uniques[appdef.MustParseQName("main.TablePlan$uniques$UniqueTable")]
		require.NotNil(u)
		cnt := 0
		for _, f := range u.Fields() {
			cnt++
			switch n := f.Name(); n {
			case "TableNumber":
				require.Equal(appdef.DataKind_int32, f.DataKind())
			default:
				require.Fail("unexpected field name", n)
			}
		}
		require.Equal(1, cnt)
	})

	// child table
	crec := appdef.CRecord(app.Type, appdef.NewQName("main", "TablePlanItem"))
	require.NotNil(crec)
	require.Equal(appdef.TypeKind_CRecord, crec.Kind())
	require.Equal(appdef.DataKind_int32, crec.Field("TableNo").DataKind())

	crec = appdef.CRecord(app.Type, appdef.NewQName("main", "NestedWithName"))
	require.NotNil(crec)
	require.True(crec.Abstract())
	field := crec.Field("ItemName")
	require.NotNil(field)
	require.Equal("Field is added to any table inherited from NestedWithName\nThe current comment is also added to scheme for this field", field.Comment())

	csingleton := appdef.CDoc(app.Type, appdef.NewQName("main", "SubscriptionProfile"))
	require.True(csingleton.Singleton())
	require.Equal("CSingletones is a configration singleton.\nThese comments are included in the statement definition, but may be overridden with `WITH Comment=...`", csingleton.Comment())

	wsingletone := appdef.WDoc(app.Type, appdef.NewQName("main", "Transaction"))
	require.True(wsingletone.Singleton())

	cmd := appdef.Command(app.Type, appdef.NewQName("main", "NewOrder"))
	require.Equal("Commands can only be declared in workspaces\nCommand can have optional argument and/or unlogged argument\nCommand can return TYPE", cmd.Comment())

	// type
	obj := appdef.Object(app.Type, appdef.NewQName("main", "SubscriptionEvent"))
	require.Equal(appdef.TypeKind_Object, obj.Kind())
	require.Equal(appdef.DataKind_string, obj.Field("Origin").DataKind())

	// view
	view := appdef.View(app.Type, appdef.NewQName("main", "XZReports"))
	require.NotNil(view)
	require.Equal(appdef.TypeKind_ViewRecord, view.Kind())
	require.Equal("VIEWs generated by the PROJECTOR.\nPrimary Key must be declared in View.", view.Comment())

	require.Equal(2, view.Value().UserFieldCount())
	require.Equal(1, view.Key().PartKey().FieldCount())
	require.Equal(4, view.Key().ClustCols().FieldCount())

	// workspace descriptor
	descr := appdef.CDoc(app.Type, appdef.NewQName("main", "MyWorkspaceDescriptor"))
	require.NotNil(descr)
	require.Equal(appdef.TypeKind_CDoc, descr.Kind())
	require.Equal(appdef.DataKind_string, descr.Field("Name").DataKind())
	require.Equal(appdef.DataKind_string, descr.Field("Country").DataKind())

	// fieldsets
	cdoc = appdef.CDoc(app.Type, appdef.NewQName("main", "WsTable"))
	require.Equal(appdef.DataKind_string, cdoc.Field("Name").DataKind())

	crec = appdef.CRecord(app.Type, appdef.NewQName("main", "Child"))
	require.Equal(appdef.DataKind_int32, crec.Field("Kind").DataKind())

	// QUERY
	q1 := appdef.Query(app.Type, appdef.NewQName("main", "Query11"))
	require.NotNil(q1)
	require.Equal(appdef.TypeKind_Query, q1.Kind())

	// CUD Projector
	proj := appdef.Projector(app.Type, appdef.NewQName("main", "RecordsRegistryProjector"))
	require.NotNil(proj)
	eventsCount := 0
	proj.Events().Enum(func(ie appdef.IProjectorEvent) {
		eventsCount++
		k, on := ie.Kind(), ie.On().QName()
		require.Len(k, 3)
		require.Contains(k, appdef.ProjectorEventKind_Insert)
		require.Contains(k, appdef.ProjectorEventKind_Activate)
		require.Contains(k, appdef.ProjectorEventKind_Deactivate)
		switch eventsCount {
		case 1:
			require.Equal(istructs.QNameCRecord, on)
		case 2:
			require.Equal(istructs.QNameWRecord, on)
		}
	})
	require.Equal(2, eventsCount)
	require.Equal(eventsCount, proj.Events().Len())

	// Execute Projector
	proj = appdef.Projector(app.Type, appdef.NewQName("main", "UpdateDashboard"))
	require.NotNil(proj)
	eventsCount = 0
	proj.Events().Enum(func(ie appdef.IProjectorEvent) {
		eventsCount++
		if eventsCount == 1 {
			require.Len(ie.Kind(), 1)
			require.Equal(appdef.ProjectorEventKind_Execute, ie.Kind()[0])
			require.Equal(appdef.NewQName("main", "NewOrder"), ie.On().QName())
		} else if eventsCount == 2 {
			require.Len(ie.Kind(), 1)
			require.Equal(appdef.ProjectorEventKind_Execute, ie.Kind()[0])
			require.Equal(appdef.NewQName("main", "NewOrder2"), ie.On().QName())
		}
	})
	require.Equal(2, eventsCount)
	require.Equal(eventsCount, proj.Events().Len())

	stateCount := 0
	for s := range proj.States().Enum {
		stateCount++
		switch stateCount {
		case 1:
			require.Equal(appdef.NewQName("sys", "AppSecret"), s.Name())
			require.Empty(s.Names())
		case 2:
			require.Equal(appdef.NewQName("sys", "Http"), s.Name())
			require.Empty(s.Names())
		default:
			require.Fail("unexpected state", "state: %v", s)
		}
	}
	require.Equal(2, stateCount)
	require.Equal(stateCount, proj.States().Len())

	intentsCount := 0
	for i := range proj.Intents().Enum {
		intentsCount++
		switch intentsCount {
		case 1:
			require.Equal(appdef.NewQName("sys", "View"), i.Name())
			require.Len(i.Names(), 4)
			require.Equal(appdef.NewQName("main", "ActiveTablePlansView"), i.Names()[0])
			require.Equal(appdef.NewQName("main", "DashboardView"), i.Names()[1])
			require.Equal(appdef.NewQName("main", "NotificationsHistory"), i.Names()[2])
			require.Equal(appdef.NewQName("main", "XZReports"), i.Names()[3])
		default:
			require.Fail("unexpected intent", "intent: %v", i)
		}
	}
	require.Equal(1, intentsCount)
	require.Equal(intentsCount, proj.Intents().Len())

	t.Run("Jobs", func(t *testing.T) {
		job1 := appdef.Job(app.Type, appdef.NewQName("main", "TestJob1"))
		require.EqualValues(`1 0 * * *`, job1.CronSchedule())
		t.Run("Job states", func(t *testing.T) {
			stateCount := 0
			for s := range proj.States().Enum {
				stateCount++
				switch stateCount {
				case 1:
					require.Equal(appdef.NewQName("sys", "AppSecret"), s.Name())
					require.Empty(s.Names())
				case 2:
					require.Equal(appdef.NewQName("sys", "Http"), s.Name())
					require.Empty(s.Names())
				default:
					require.Fail("unexpected state", "storage: %v", s.Name())
				}
			}
			require.Equal(2, stateCount)
		})

		job2 := appdef.Job(app.Type, appdef.NewQName("main", "TestJob2"))
		require.EqualValues(`@every 2m30s`, job2.CronSchedule())
	})

	cmd = appdef.Command(app.Type, appdef.NewQName("main", "NewOrder2"))
	require.Equal(1, cmd.States().Len())
	require.NotNil(cmd.States().Storage(appdef.NewQName("sys", "AppSecret")))
	require.Equal(1, cmd.States().Len())
	intent := cmd.Intents().Storage(appdef.NewQName("sys", "Record"))
	require.NotNil(intent)
	require.True(intent.Names().Contains(appdef.NewQName("main", "Transaction")))

	localNames := app.PackageLocalNames()
	require.Len(localNames, 4)
	require.Contains(localNames, appdef.SysPackage)
	require.Contains(localNames, "main")
	require.Contains(localNames, "air")
	require.Contains(localNames, "untill")

	require.Equal(appdef.SysPackagePath, app.PackageFullPath(appdef.SysPackage))
	require.Equal("github.com/untillpro/main", app.PackageFullPath("main"))
	require.Equal("github.com/untillpro/airsbp", app.PackageFullPath("air"))
	require.Equal("github.com/untillpro/untill", app.PackageFullPath("untill"))

	require.Equal("main", app.PackageLocalName("github.com/untillpro/main"))
	require.Equal("air", app.PackageLocalName("github.com/untillpro/airsbp"))
	require.Equal("untill", app.PackageLocalName("github.com/untillpro/untill"))
}

type ParserAssertions struct {
	*require.Assertions
}

func (require *ParserAssertions) AppSchemaError(sql string, expectErrors ...string) {
	_, err := require.AppSchema(sql)
	require.EqualError(err, strings.Join(expectErrors, "\n"))
}

func (require *ParserAssertions) NoAppSchemaError(sql string) {
	_, err := require.AppSchema(sql)
	require.NoError(err)
}

func (require *ParserAssertions) PkgSchema(filename, pkg, sql string) *PackageSchemaAST {
	ast, err := ParseFile(filename, sql)
	require.NoError(err)
	p, err := BuildPackageSchema(pkg, []*FileSchemaAST{ast})
	require.NoError(err)
	return p
}

func (require *ParserAssertions) AppSchema(sql string) (*AppSchemaAST, error) {
	pkg := require.PkgSchema("file.vsql", "github.com/company/pkg", sql)
	return BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
}

func assertions(t *testing.T) *ParserAssertions {
	return &ParserAssertions{require.New(t)}
}

func Test_Refs_NestedTables(t *testing.T) {

	require := require.New(t)

	fs, err := ParseFile("file1.vsql", `APPLICATION test();
	WORKSPACE MyWorkspace(
		TABLE table1 INHERITS sys.CDoc (
			items TABLE inner1 (
				table1 ref,
				ref1 ref(table3),
				urg_number int32
			)
		);
		TABLE table2 INHERITS sys.CRecord (
		);
		TABLE table3 INHERITS sys.CDoc (
			items table2
		);
	);
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("test/pkg1", []*FileSchemaAST{fs})
	require.NoError(err)

	packages, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	adb := appdef.New()
	require.NoError(BuildAppDefs(packages, adb))

	app, err := adb.Build()
	require.NoError(err)

	inner1 := app.Type(appdef.NewQName("pkg1", "inner1"))
	ref1 := inner1.(appdef.IFields).RefField("ref1")
	require.EqualValues(appdef.QNames{appdef.NewQName("pkg1", "table3")}, ref1.Refs())
}

func Test_CircularReferencesTables(t *testing.T) {
	require := require.New(t)
	// Tables
	fs, err := ParseFile("file1.vsql", `APPLICATION test();
	WORKSPACE MyWorkspace(
		ABSTRACT TABLE table2 INHERITS table2 ();
		ABSTRACT TABLE table3 INHERITS table3 ();
		ABSTRACT TABLE table4 INHERITS table5 ();
		ABSTRACT TABLE table5 INHERITS table6 ();
		ABSTRACT TABLE table6 INHERITS table4 ();
	);
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("pkg/test", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.EqualError(err, strings.Join([]string{
		"file1.vsql:3:3: circular reference in INHERITS",
		"file1.vsql:4:3: circular reference in INHERITS",
		"file1.vsql:5:3: circular reference in INHERITS",
		"file1.vsql:6:3: circular reference in INHERITS",
		"file1.vsql:7:3: circular reference in INHERITS",
	}, "\n"))
}

func Test_CircularReferencesWorkspaces(t *testing.T) {
	require := require.New(t)
	// Workspaces
	fs, err := ParseFile("file1.vsql", `APPLICATION test();
	ABSTRACT WORKSPACE w1();
		ABSTRACT WORKSPACE w2 INHERITS w1,w2(
			TABLE table4 INHERITS sys.CDoc();
		);
		ABSTRACT WORKSPACE w3 INHERITS w4();
		ABSTRACT WORKSPACE w4 INHERITS w5();
		ABSTRACT WORKSPACE w5 INHERITS w3();
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("pkg/test", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})

	require.EqualError(err, strings.Join([]string{
		"file1.vsql:3:37: circular reference in INHERITS",
		"file1.vsql:6:34: circular reference in INHERITS",
		"file1.vsql:7:34: circular reference in INHERITS",
		"file1.vsql:8:34: circular reference in INHERITS",
	}, "\n"))
}

func Test_Workspace_Defs(t *testing.T) {

	require := require.New(t)

	fs1, err := ParseFile("file1.vsql", `APPLICATION test();
		ABSTRACT WORKSPACE AWorkspace(
			TABLE table1 INHERITS sys.CDoc (
				a ref,
				items TABLE inner1 (
					b ref
				)
			);
		);
	`)
	require.NoError(err)
	fs2, err := ParseFile("file2.vsql", `
		ALTER WORKSPACE AWorkspace(
			TABLE table2 INHERITS sys.CDoc (
				a ref,
				items TABLE inner2 (
					b ref
				)
			);
		);
		WORKSPACE MyWorkspace INHERITS AWorkspace();
		WORKSPACE MyWorkspace2 INHERITS AWorkspace();
		ALTER WORKSPACE sys.Profile(
			USE WORKSPACE MyWorkspace;
		);
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("test/pkg1", []*FileSchemaAST{fs1, fs2})
	require.NoError(err)

	packages, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	builder := appdef.New()
	require.NoError(BuildAppDefs(packages, builder))

	app, err := builder.Build()
	require.NoError(err)

	ws := app.Workspace(appdef.NewQName("pkg1", "MyWorkspace"))

	require.Equal(appdef.TypeKind_CDoc, ws.Type(appdef.NewQName("pkg1", "table1")).Kind())
	require.Equal(appdef.TypeKind_CDoc, ws.Type(appdef.NewQName("pkg1", "table2")).Kind())
	require.Equal(appdef.TypeKind_CRecord, ws.Type(appdef.NewQName("pkg1", "inner1")).Kind())
	require.Equal(appdef.TypeKind_CRecord, ws.Type(appdef.NewQName("pkg1", "inner2")).Kind())
	require.Equal(appdef.TypeKind_Command, ws.Type(appdef.NewQName("sys", "CreateLogin")).Kind())

	wsProfile := app.Workspace(appdef.NewQName("sys", "Profile"))

	require.Equal(appdef.TypeKind_Workspace, wsProfile.Type(appdef.NewQName("pkg1", "MyWorkspace")).Kind())
	require.Equal(appdef.NullType, wsProfile.Type(appdef.NewQName("pkg1", "MyWorkspace2")))
}

func Test_Workspace_Defs3(t *testing.T) {
	require := require.New(t)
	fs, err := ParseFile("file1.vsql", `IMPORT SCHEMA 'test/pkg1';
		APPLICATION test(
			USE pkg1;
		);
		WORKSPACE Workspace2 INHERITS pkg1.Workspace1 (
			TABLE Table2 INHERITS sys.CDoc (
				pkg1.Type1
			);
		);
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("test/pkg2", []*FileSchemaAST{fs})
	require.NoError(err)

	fs2, err := ParseFile("file2.vsql", `
		ABSTRACT WORKSPACE Workspace1(
			TYPE Type1 ();
		);
	`)
	require.NoError(err)
	pkg2, err := BuildPackageSchema("test/pkg1", []*FileSchemaAST{fs2})
	require.NoError(err)

	packages, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
		pkg2,
	})
	require.NoError(err)

	builder := appdef.New()
	require.NoError(BuildAppDefs(packages, builder))

	_, err = builder.Build()
	require.NoError(err)
}

func Test_Alter_Workspace(t *testing.T) {

	require := assertions(t)

	t.Run("Try alter non-alterable workspace", func(t *testing.T) {
		pkg0 := require.PkgSchema("file0.vsql", "org/main", `
		IMPORT SCHEMA 'org/pkg1';
		IMPORT SCHEMA 'org/pkg2';
		APPLICATION test(
			USE pkg1;
			USE pkg2;
		);
		`)
		pkg1 := require.PkgSchema("file1.vsql", "org/pkg1", `
			ABSTRACT WORKSPACE AWorkspace(
				TABLE table1 INHERITS sys.CDoc (a ref);
			);
		`)
		pkg2 := require.PkgSchema("file2.vsql", "org/pkg2", `
			IMPORT SCHEMA 'org/pkg1'
			ALTER WORKSPACE pkg1.AWorkspace(
				TABLE table2 INHERITS sys.CDoc (a ref);
			);
		`)

		_, err := BuildAppSchema([]*PackageSchemaAST{
			getSysPackageAST(),
			pkg0,
			pkg1,
			pkg2,
		})
		require.EqualError(err, strings.Join([]string{
			"file2.vsql:3:20: workspace pkg1.AWorkspace is not alterable",
		}, "\n"))
	})

	t.Run("Alter workspace in a different package", func(t *testing.T) {
		pkg0 := require.PkgSchema("file0.vsql", "org/main", `
		IMPORT SCHEMA 'org/pkg1';
		IMPORT SCHEMA 'org/pkg2';
		APPLICATION test(
			USE pkg1;
			USE pkg2;
		);
		`)
		pkg1 := require.PkgSchema("file1.vsql", "org/pkg1", `
			ALTERABLE WORKSPACE AWorkspace(
			);
		`)
		pkg2 := require.PkgSchema("file2.vsql", "org/pkg2", `
			IMPORT SCHEMA 'org/pkg1'
			ALTER WORKSPACE pkg1.AWorkspace(
				TABLE table2 INHERITS sys.CDoc (a ref);
			);
		`)

		schema, err := BuildAppSchema([]*PackageSchemaAST{
			getSysPackageAST(),
			pkg0,
			pkg1,
			pkg2,
		})
		require.NoError(err)
		builder := appdef.New()
		err = BuildAppDefs(schema, builder)
		require.NoError(err)
	})
	t.Run("Alter workspace in sys package", func(t *testing.T) {
		schema, err := require.AppSchema(`APPLICATION SomeApp();
		ALTER WORKSPACE sys.AppWorkspaceWS (
			TYPE SomeType (
				field int32
			);
		)
		`)
		require.NoError(err)
		builder := appdef.New()
		err = BuildAppDefs(schema, builder)
		require.NoError(err)
	})
}

func Test_DupFieldsInTypes(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("file1.vsql", `APPLICATION test();
	WORKSPACE MyWorkspace(
		TYPE RootType (
			Id int32
		);
		TYPE BaseType(
			RootType,
			baseField int
		);
		TYPE BaseType2 (
			someField int
		);
		TYPE MyType(
			BaseType,
			BaseType2,
			field varchar,
			field varchar,
			baseField varchar,
			someField int,
			Id varchar
		)
	)
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("pkg/test", []*FileSchemaAST{fs})
	require.NoError(err)

	packages, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	err = BuildAppDefs(packages, appdef.New())
	require.EqualError(err, strings.Join([]string{
		"file1.vsql:17:4: redefinition of field",
		"file1.vsql:18:4: redefinition of baseField",
		"file1.vsql:19:4: redefinition of someField",
		"file1.vsql:20:4: redefinition of Id",
	}, "\n"))

}

func Test_Varchar(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("file1.vsql", fmt.Sprintf(`APPLICATION test(); WORKSPACE MyWorkspace(
	TYPE RootType (
		Oversize varchar(%d)
	);
	TYPE CDoc1 (
		Oversize varchar(%d)
	););
	`, uint32(appdef.MaxFieldLength)+1, uint32(appdef.MaxFieldLength)+1))
	require.NoError(err)
	pkg, err := BuildPackageSchema("pkg/test", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.EqualError(err, strings.Join([]string{
		fmt.Sprintf("file1.vsql:3:12: maximum field length is %d", appdef.MaxFieldLength),
		fmt.Sprintf("file1.vsql:6:12: maximum field length is %d", appdef.MaxFieldLength),
	}, "\n"))

}

func Test_DupFieldsInTables(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("file1.vsql", `APPLICATION test(); WORKSPACE MyWorkspace(
	TYPE RootType (
		Kind int32
	);
	TYPE BaseType(
		RootType,
		baseField int
	);
	TYPE BaseType2 (
		someField int
	);
	ABSTRACT TABLE ByBaseTable INHERITS sys.CDoc (
		Name varchar,
		Code varchar
	);
	TABLE MyTable INHERITS ByBaseTable(
		BaseType,
		BaseType2,
		newField varchar,
		field varchar,
		field varchar, 		-- duplicated in the this table
		baseField varchar,		-- duplicated in the first OF
		someField int,		-- duplicated in the second OF
		Kind int,			-- duplicated in the first OF (2nd level)
		Name int,			-- duplicated in the inherited table
		ID varchar
))
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("pkg/test", []*FileSchemaAST{fs})
	require.NoError(err)

	packages, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	err = BuildAppDefs(packages, appdef.New())
	require.EqualError(err, strings.Join([]string{
		"file1.vsql:21:3: redefinition of field",
		"file1.vsql:22:3: redefinition of baseField",
		"file1.vsql:23:3: redefinition of someField",
		"file1.vsql:24:3: redefinition of Kind",
		"file1.vsql:25:3: redefinition of Name",
	}, "\n"))

}

func Test_AbstractTables(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("file1.vsql", `APPLICATION test();

	WORKSPACE MyWorkspace1(
	TABLE ByBaseTable INHERITS sys.CDoc (
		Name varchar
	);
	TABLE MyTable INHERITS ByBaseTable(		-- NOT ALLOWED (base table must be abstract)
	);

	TABLE My1 INHERITS sys.CRecord(
		f1 ref(AbstractTable)				-- NOT ALLOWED (reference to abstract table)
	);

	ABSTRACT TABLE AbstractTable INHERITS sys.CDoc(
	);

	EXTENSION ENGINE BUILTIN (

			PROJECTOR proj1
            AFTER INSERT ON AbstractTable 	-- NOT ALLOWED (projector refers to abstract table)
            INTENTS(sys.SendMail);

			SYNC PROJECTOR proj2
            AFTER INSERT ON My1
            INTENTS(sys.Record(AbstractTable));	-- NOT ALLOWED (projector refers to abstract table)

			PROJECTOR proj3
            AFTER INSERT ON My1
			STATE(sys.Record(AbstractTable))		-- NOT ALLOWED (projector refers to abstract table)
            INTENTS(sys.SendMail);
		);
		TABLE My2 INHERITS sys.CRecord(
			nested AbstractTable			-- NOT ALLOWED
		);
		TABLE My3 INHERITS sys.CRecord(
			f int,
			items ABSTRACT TABLE Nested()	-- NOT ALLOWED
		);
	)
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("test/pkg1", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.EqualError(err, strings.Join([]string{
		"file1.vsql:7:25: base table must be abstract",
		"file1.vsql:20:29: projector refers to abstract table AbstractTable",
		"file1.vsql:25:21: projector refers to abstract table AbstractTable",
		"file1.vsql:29:10: projector refers to abstract table AbstractTable",
		"file1.vsql:33:11: nested abstract table AbstractTable",
		"file1.vsql:37:4: nested abstract table Nested",
		"file1.vsql:11:10: reference to abstract table AbstractTable",
	}, "\n"))

}

func Test_AbstractTables2(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("file1.vsql", `APPLICATION test();
	WORKSPACE MyWorkspace1(
		ABSTRACT TABLE AbstractTable INHERITS sys.CDoc(
		);

		TABLE My2 INHERITS sys.CRecord(
			nested AbstractTable			-- NOT ALLOWED
		);
	);
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("test/pkg", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.EqualError(err, strings.Join([]string{
		"file1.vsql:7:11: nested abstract table AbstractTable",
	}, "\n"))

}

func Test_WorkspaceDescriptors(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("file1.vsql", `APPLICATION test();
	ROLE R1;
	WORKSPACE W1(
		DESCRIPTOR(); -- gets name W1Descriptor
	);
	WORKSPACE W2(
		DESCRIPTOR W2D(); -- gets name W2D
	);
	WORKSPACE W3(
		DESCRIPTOR R1(); -- duplicated name
	);
	ROLE W2D; -- duplicated name
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("test/pkg", []*FileSchemaAST{fs})
	require.EqualError(err, strings.Join([]string{
		"file1.vsql:10:14: redefinition of R1",
		"file1.vsql:12:2: redefinition of W2D",
	}, "\n"))

	require.Equal(Ident("W1Descriptor"), pkg.Ast.Statements[2].Workspace.Descriptor.Name)
	require.Equal(Ident("W2D"), pkg.Ast.Statements[3].Workspace.Descriptor.Name)
}
func Test_PanicUnknownFieldType(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("file1.vsql", `APPLICATION test(); WORKSPACE MyWorkspace(
	TABLE MyTable INHERITS sys.CDoc (
		Name asdasd,
		Code varchar
	))
	`)
	require.NoError(err)
	pkg, err := BuildPackageSchema("test/pkg", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.EqualError(err, strings.Join([]string{
		"file1.vsql:3:8: undefined data type or table: asdasd",
	}, "\n"))

}

func Test_Expressions(t *testing.T) {
	require := require.New(t)

	_, err := ParseFile("file1.vsql", `WORKSPACE MyWorkspace(
	TABLE MyTable(
		Int1 varchar DEFAULT 1 CHECK(Int1 > Int2),
		Int1 int DEFAULT 1 CHECK(Text != 'asd'),
		Int1 int DEFAULT 1 CHECK(Int2 > -5),
		Int1 int DEFAULT 1 CHECK(TextField > 'asd' AND (SomeFloat/3.2)*4 != 5.003),
		Int1 int DEFAULT 1 CHECK(SomeFunc('a', TextField) AND BoolField=FALSE),

		CHECK(MyRowValidator(this))
))
	`)
	require.NoError(err)

}

func Test_Duplicates(t *testing.T) {
	require := require.New(t)

	ast1, err := ParseFile("file1.vsql", `APPLICATION test();
	EXTENSION ENGINE BUILTIN (
		FUNCTION MyTableValidator() RETURNS void;
		FUNCTION MyTableValidator(TableRow) RETURNS string;
		FUNCTION MyFunc2() RETURNS void;
	);
	`)
	require.NoError(err)

	ast2, err := ParseFile("file2.vsql", `
	WORKSPACE ChildWorkspace (
		TABLE Rec1 INHERITS CRecord();
		TAG MyFunc2; -- redeclared
		EXTENSION ENGINE BUILTIN (
			FUNCTION MyFunc3() RETURNS void;
			FUNCTION MyFunc4() RETURNS void;
		);
		WORKSPACE InnerWorkspace (
			ROLE MyFunc4; -- redeclared
		);
		TABLE Doc1 INHERITS sys.ODoc(
			nested1 Rec1,
			nested2 TABLE Rec1() -- redeclared
		)
	)
	`)
	require.NoError(err)

	_, err = BuildPackageSchema("test/pkg", []*FileSchemaAST{ast1, ast2})

	require.EqualError(err, strings.Join([]string{
		"file1.vsql:4:3: redefinition of MyTableValidator",
		"file2.vsql:4:3: redefinition of MyFunc2",
		"file2.vsql:10:4: redefinition of MyFunc4",
		"file2.vsql:14:12: redefinition of Rec1",
	}, "\n"))

}

func Test_DuplicatesInViews(t *testing.T) {
	require := require.New(t)

	ast, err := ParseFile("file2.vsql", `APPLICATION test();
	WORKSPACE Workspace (
		VIEW test(
			field1 int,
			field2 int,
			field1 varchar,
			PRIMARY KEY(field1),
			PRIMARY KEY(field2)
		) AS RESULT OF Proj1;

		EXTENSION ENGINE BUILTIN (
			PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
			COMMAND Orders()
		);
	)
	`)
	require.NoError(err)

	pkg, err := BuildPackageSchema("test/pkg", []*FileSchemaAST{ast})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		pkg,
		getSysPackageAST(),
	})

	require.EqualError(err, strings.Join([]string{
		"file2.vsql:6:4: redefinition of field1",
		"file2.vsql:8:16: redefinition of primary key",
	}, "\n"))

}
func Test_Views(t *testing.T) {
	require := assertions(t)

	require.AppSchemaError(`APPLICATION test(); WORKSPACE Workspace (
			VIEW test(
				field1 int,
				PRIMARY KEY(field2)
			) AS RESULT OF Proj1;
			EXTENSION ENGINE BUILTIN (
				PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
				COMMAND Orders()
			);
			)
	`, "file.vsql:4:17: undefined field field2")

	require.AppSchemaError(`APPLICATION test(); WORKSPACE Workspace (
			VIEW test(
				field1 varchar,
				PRIMARY KEY((field1))
			) AS RESULT OF Proj1;
			EXTENSION ENGINE BUILTIN (
				PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
				COMMAND Orders()
			);
			)
	`, "file.vsql:4:18: varchar field field1 not supported in partition key")

	require.AppSchemaError(`APPLICATION test(); WORKSPACE Workspace (
		VIEW test(
			field1 bytes,
			PRIMARY KEY((field1))
		) AS RESULT OF Proj1;
		EXTENSION ENGINE BUILTIN (
			PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
			COMMAND Orders()
		);
	)
	`, "file.vsql:4:17: bytes field field1 not supported in partition key")

	require.AppSchemaError(`APPLICATION test(); WORKSPACE Workspace (
		VIEW test(
			field1 varchar,
			field2 int,
			PRIMARY KEY(field1, field2)
		) AS RESULT OF Proj1;
		EXTENSION ENGINE BUILTIN (
			PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
			COMMAND Orders()
		);
	)
	`, "file.vsql:5:16: varchar field field1 can only be the last one in clustering key")

	require.AppSchemaError(`APPLICATION test(); WORKSPACE Workspace (
		VIEW test(
			field1 bytes,
			field2 int,
			PRIMARY KEY(field1, field2)
		) AS RESULT OF Proj1;
		EXTENSION ENGINE BUILTIN (
			PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
			COMMAND Orders()
		);
	)
	`, "file.vsql:5:16: bytes field field1 can only be the last one in clustering key")

	require.AppSchemaError(`APPLICATION test(); WORKSPACE Workspace (
		ABSTRACT TABLE abc INHERITS sys.CDoc();
		VIEW test(
			field1 ref(abc),
			field2 ref(unexisting),
			PRIMARY KEY(field1, field2)
		) AS RESULT OF Proj1;
		EXTENSION ENGINE BUILTIN (
			PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
			COMMAND Orders()
		);
	)
	`, "file.vsql:4:15: reference to abstract table abc", "file.vsql:5:15: undefined table: unexisting")

	require.AppSchemaError(`APPLICATION test(); WORKSPACE Workspace (
		VIEW test(
			fld1 int32
		) AS RESULT OF Proj1;
		EXTENSION ENGINE BUILTIN (
			PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
			COMMAND Orders()
		);
	)
	`, "file.vsql:2:3: primary key not defined")

	t.Run("record field in partition key", func(t *testing.T) {
		require.AppSchemaError(`APPLICATION test(); WORKSPACE Workspace (
			VIEW test(
				i int32,
				field1 record,
				PRIMARY KEY((field1))
			) AS RESULT OF Proj1;
			EXTENSION ENGINE BUILTIN (
				PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
				COMMAND Orders()
			);
			)
		`, "file.vsql:5:18: record field field1 not supported in partition key")
	})

	t.Run("record field in clustering key", func(t *testing.T) {
		require.AppSchemaError(`APPLICATION test(); WORKSPACE Workspace (
			VIEW test(
				i int32,
				field1 record,
				PRIMARY KEY((i), field1)
			) AS RESULT OF Proj1;
			EXTENSION ENGINE BUILTIN (
				PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
				COMMAND Orders()
			);
			)
		`, "file.vsql:5:22: record field field1 not supported in partition key")
	})

	t.Run("record fields", func(t *testing.T) {
		schema, err := require.AppSchema(`APPLICATION test(); WORKSPACE Workspace (
			VIEW test(
				i int32,
				j int32,
				field1 record,
				PRIMARY KEY((i), j)
			) AS RESULT OF Proj1;
			EXTENSION ENGINE BUILTIN (
				PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
				COMMAND Orders()
			);
			)
		`)
		require.NoError(err)
		require.NotNil(schema)
		adf := appdef.New()
		require.NoError(BuildAppDefs(schema, adf))
		def, err := adf.Build()

		require.NoError(err)
		view := appdef.View(def.Type, appdef.NewQName("pkg", "test"))
		require.NotNil(view)
		require.NotNil(view.Value().Field("field1"))
	})
}

func Test_Views2(t *testing.T) {
	require := require.New(t)

	{
		ast, err := ParseFile("file2.vsql", `APPLICATION test(); WORKSPACE Workspace (
			VIEW test(
				-- comment1
				field1 int,
				-- comment2
				field2 varchar(20),
				-- comment3
				field3 bytes(20),
				-- comment4
				field4 ref,
				PRIMARY KEY((field1,field4),field2)
			) AS RESULT OF Proj1;
			EXTENSION ENGINE BUILTIN (
				PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
				COMMAND Orders()
			);
		)
		`)
		require.NoError(err)
		pkg, err := BuildPackageSchema("test", []*FileSchemaAST{ast})
		require.NoError(err)

		packages, err := BuildAppSchema([]*PackageSchemaAST{
			getSysPackageAST(),
			pkg,
		})
		require.NoError(err)

		appBld := appdef.New()
		err = BuildAppDefs(packages, appBld)
		require.NoError(err)

		app, err := appBld.Build()
		require.NoError(err)

		v := appdef.View(app.Type, appdef.NewQName("test", "test"))
		require.NotNil(v)
	}
	{
		ast, err := ParseFile("file2.vsql", `APPLICATION test(); WORKSPACE Workspace (
			VIEW test(
				-- comment1
				field1 int,
				-- comment2
				field3 bytes(20),
				-- comment4
				field4 ref,
				PRIMARY KEY((field1),field4,field3)
			) AS RESULT OF Proj1;
			EXTENSION ENGINE BUILTIN (
				PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
				COMMAND Orders()
			);
		)
		`)
		require.NoError(err)
		pkg, err := BuildPackageSchema("test", []*FileSchemaAST{ast})
		require.NoError(err)

		packages, err := BuildAppSchema([]*PackageSchemaAST{
			getSysPackageAST(),
			pkg,
		})
		require.NoError(err)

		appBld := appdef.New()
		err = BuildAppDefs(packages, appBld)
		require.NoError(err)

		app, err := appBld.Build()
		require.NoError(err)

		v := appdef.View(app.Type, appdef.NewQName("test", "test"))
		require.NotNil(v)
	}
	{
		ast, err := ParseFile("file2.vsql", `APPLICATION test(); WORKSPACE Workspace (
			VIEW test(
				-- comment1
				field1 int,
				-- comment2
				field3 bytes(20),
				-- comment4
				field4 ref,
				PRIMARY KEY((field1),field4,field3)
			) AS RESULT OF Proj1;
			EXTENSION ENGINE BUILTIN (
				PROJECTOR Proj1 AFTER EXECUTE ON (Orders);
				COMMAND Orders()
			);
		)
		`)
		require.NoError(err)
		pkg, err := BuildPackageSchema("test", []*FileSchemaAST{ast})
		require.NoError(err)

		_, err = BuildAppSchema([]*PackageSchemaAST{
			getSysPackageAST(),
			pkg,
		})
		require.Error(err, "file2.vsql:2:4: projector Proj1 does not declare intent for view test")

	}

}
func Test_Comments(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.vsql", `
	EXTENSION ENGINE BUILTIN (

	-- My function
	-- line 2
	FUNCTION MyFunc() RETURNS void;

	/* 	Multiline
		comment  */
	FUNCTION MyFunc1() RETURNS void;
	);

	`)
	require.NoError(err)

	ps, err := BuildPackageSchema("test", []*FileSchemaAST{fs})
	require.NoError(err)

	require.NotNil(ps.Ast.Statements[0].ExtEngine.Statements[0].Function.Comments)

	comments := ps.Ast.Statements[0].ExtEngine.Statements[0].Function.GetComments()
	require.Len(comments, 2)
	require.Equal("My function", comments[0])
	require.Equal("line 2", comments[1])

	fn := ps.Ast.Statements[0].ExtEngine.Statements[1].Function
	comments = fn.GetComments()
	require.Len(comments, 2)
	require.Equal("Multiline", comments[0])
	require.Equal("comment", comments[1])
}

func Test_Undefined(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.vsql", `APPLICATION test();
	WORKSPACE test (
		EXTENSION ENGINE WASM (
			COMMAND Orders() WITH Tags=(UndefinedTag);
			PROJECTOR ImProjector AFTER EXECUTE ON xyz.CreateUPProfile;
			COMMAND CmdFakeReturn() RETURNS text;
			COMMAND CmdNoReturn() RETURNS void;
			COMMAND CmdFakeArg(text);
			COMMAND CmdVoidArg(void);
			COMMAND CmdFakeUnloggedArg(UNLOGGED text);
		)
	)
	`)
	require.NoError(err)

	pkg, err := BuildPackageSchema("test", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{pkg, getSysPackageAST()})

	require.EqualError(err, strings.Join([]string{
		"example.vsql:4:32: undefined tag: UndefinedTag",
		"example.vsql:5:43: xyz undefined",
		"example.vsql:6:36: undefined type or table: text",
		"example.vsql:8:23: undefined type or table: text",
		"example.vsql:10:40: undefined type or table: text",
	}, "\n"))
}

func Test_Projectors(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.vsql", `APPLICATION test();
	WORKSPACE test (
		TABLE Order INHERITS sys.ODoc();
		EXTENSION ENGINE WASM (
			COMMAND Orders();
			PROJECTOR ImProjector1 AFTER EXECUTE ON test.CreateUPProfile; 			-- Undefined
			PROJECTOR ImProjector2 AFTER EXECUTE ON Order; 							-- Bad: Order is not a type or command
			PROJECTOR ImProjector3 AFTER UPDATE ON Order; 				-- Bad
			PROJECTOR ImProjector4 AFTER ACTIVATE ON Order; 			-- Bad
			PROJECTOR ImProjector5 AFTER DEACTIVATE ON Order; 			-- Bad
			PROJECTOR ImProjector6 AFTER INSERT ON Order OR AFTER EXECUTE ON Orders;	-- Good
			PROJECTOR ImProjector7 AFTER EXECUTE WITH PARAM ON Bill;	-- Bad: Type undefined
			PROJECTOR ImProjector8 AFTER EXECUTE WITH PARAM ON sys.ODoc;	-- Good
			PROJECTOR ImProjector9 AFTER EXECUTE WITH PARAM ON sys.ORecord;	-- Bad
		);
	)
	`)
	require.NoError(err)

	pkg, err := BuildPackageSchema("test", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{pkg, getSysPackageAST()})

	require.EqualError(err, strings.Join([]string{
		"example.vsql:6:44: undefined command: test.CreateUPProfile",
		"example.vsql:7:44: undefined command: Order",
		"example.vsql:8:43: only INSERT allowed for ODoc or ORecord",
		"example.vsql:9:45: only INSERT allowed for ODoc or ORecord",
		"example.vsql:10:47: only INSERT allowed for ODoc or ORecord",
		"example.vsql:12:55: undefined type or ODoc: Bill",
		"example.vsql:14:55: undefined type or ODoc: sys.ORecord",
	}, "\n"))
}

func Test_Imports(t *testing.T) {
	require := assertions(t)

	pkg1 := require.PkgSchema("example.vsql", "github.com/untillpro/airsbp3/pkg1", `
	IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg2';
	IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg3' AS air;
	APPLICATION test(
		USE pkg2;
		USE pkg3;
	);
	ABSTRACT WORKSPACE base INHERITS pkg2.BaseWorkspace (
		TAG SomeTag;
	);
	WORKSPACE test INHERITS base (
		EXTENSION ENGINE WASM (
    		COMMAND Orders1 WITH Tags=(pkg2.InheritedTag); -- pkg2.InheritedTag undefined
    		COMMAND Orders2 WITH Tags=(pkg3.SomePkg3Tag); -- pkg3.SomePkg3Tag undefined
    		QUERY Query3 RETURNS void WITH Tags=(air.UnknownTag); -- air.UnknownTag undefined
    		PROJECTOR ImProjector AFTER EXECUTE ON Air.CreateUPProfil; -- Air undefined
    		COMMAND Orders3 WITH Tags=(SomeTag); -- SomeTag undefined
		)
	)
	`)

	pkg2 := require.PkgSchema("example.vsql", "github.com/untillpro/airsbp3/pkg2", `
	ABSTRACT WORKSPACE BaseWorkspace(
		TAG InheritedTag;
	);
	`)

	pkg3 := require.PkgSchema("example.vsql", "github.com/untillpro/airsbp3/pkg3", `TAG SomePkg3Tag;`)

	_, err := BuildAppSchema([]*PackageSchemaAST{getSysPackageAST(), pkg1, pkg2, pkg3})
	require.EqualError(err, strings.Join([]string{
		"example.vsql:13:34: undefined tag: pkg2.InheritedTag",
		"example.vsql:14:34: undefined tag: pkg3.SomePkg3Tag",
		"example.vsql:15:44: undefined tag: air.UnknownTag",
		"example.vsql:16:46: Air undefined",
		"example.vsql:17:34: undefined tag: SomeTag",
	}, "\n"))

}

func Test_AbstractWorkspace(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.vsql", `APPLICATION test();
	WORKSPACE ws1 ();
	ABSTRACT WORKSPACE ws2(
		DESCRIPTOR(					-- Incorrect
			a int
		);
	);
	WORKSPACE ws4 INHERITS ws2 ();
	WORKSPACE ws5 INHERITS ws1 ();  -- Incorrect
	`)
	require.NoError(err)

	ps, err := BuildPackageSchema("test", []*FileSchemaAST{fs})
	require.NoError(err)

	require.False(ps.Ast.Statements[1].Workspace.Abstract)
	require.True(ps.Ast.Statements[2].Workspace.Abstract)
	require.False(ps.Ast.Statements[3].Workspace.Abstract)
	require.Equal("ws2", ps.Ast.Statements[3].Workspace.Inherits[0].String())

	_, err = BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		ps,
	})
	require.EqualError(err, strings.Join([]string{
		"example.vsql:4:13: abstract workspace cannot have a descriptor",
		"example.vsql:9:25: base workspace must be abstract",
	}, "\n"))

}

func Test_UniqueFields(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.vsql", `APPLICATION test(); WORKSPACE MyWorkspace(
	TABLE MyTable INHERITS sys.CDoc (
		Int1 int32,
		Int2 int32 NOT NULL,
		UNIQUEFIELD Int1,
		UNIQUEFIELD Int2
))
	`)
	require.NoError(err)

	pkg, err := BuildPackageSchema("test", []*FileSchemaAST{fs})
	require.NoError(err)

	packages, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	appBld := appdef.New()
	err = BuildAppDefs(packages, appBld)
	require.NoError(err)

	app, err := appBld.Build()
	require.NoError(err)

	cdoc := appdef.CDoc(app.Type, appdef.NewQName("test", "MyTable"))
	require.NotNil(cdoc)

	fld := cdoc.UniqueField()
	require.NotNil(fld)
	require.Equal("Int2", fld.Name())
}

func Test_NestedTables(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.vsql", `APPLICATION test(); WORKSPACE MyWorkspace(
	TABLE NestedTable INHERITS sys.CRecord (
		ItemName varchar,
		DeepNested TABLE DeepNestedTable (
			ItemName varchar
		)
	))
	`)
	require.NoError(err)

	pkg, err := BuildPackageSchema("test", []*FileSchemaAST{fs})
	require.NoError(err)

	packages, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	appBld := appdef.New()
	err = BuildAppDefs(packages, appBld)
	require.NoError(err)

	app, err := appBld.Build()
	require.NoError(err)

	require.NotNil(appdef.CRecord(app.Type, appdef.NewQName("test", "NestedTable")))
	require.NotNil(appdef.CRecord(app.Type, appdef.NewQName("test", "DeepNestedTable")))
}

func Test_SemanticAnalysisForReferences(t *testing.T) {
	t.Run("Should return error because CDoc references to ODoc", func(t *testing.T) {
		require := require.New(t)

		fs, err := ParseFile("example.vsql", `APPLICATION test(); WORKSPACE MyWorkspace(
		TABLE OTable INHERITS sys.ODoc ();
		TABLE CTable INHERITS sys.CDoc (
			OTableRef ref(OTable)
		))
		`)
		require.NoError(err)

		pkg, err := BuildPackageSchema("test", []*FileSchemaAST{fs})
		require.NoError(err)

		packages, err := BuildAppSchema([]*PackageSchemaAST{
			getSysPackageAST(),
			pkg,
		})
		require.NoError(err)

		appBld := appdef.New()
		err = BuildAppDefs(packages, appBld)
		require.Error(err)
		require.Contains(err.Error(), "table test.CTable can not reference to ODoc «test.OTable»")
	})
}

func Test_1KStringField(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.vsql", `APPLICATION test(); WORKSPACE MyWorkspace(
	TABLE MyTable INHERITS sys.CDoc (
		KB varchar(1024)
))
	`)
	require.NoError(err)

	pkg, err := BuildPackageSchema("test", []*FileSchemaAST{fs})
	require.NoError(err)

	packages, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	appBld := appdef.New()
	err = BuildAppDefs(packages, appBld)
	require.NoError(err)

	app, err := appBld.Build()
	require.NoError(err)

	cdoc := appdef.CDoc(app.Type, appdef.NewQName("test", "MyTable"))
	require.NotNil(cdoc)

	fld := cdoc.Field("KB")
	require.NotNil(fld)

	cnt := 0
	for _, c := range fld.Constraints() {
		cnt++
		require.Equal(appdef.ConstraintKind_MaxLen, c.Kind())
		require.EqualValues(1024, c.Value())
	}
	require.Equal(1, cnt)
}

func Test_ReferenceToNoTable(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.vsql", `APPLICATION test(); WORKSPACE MyWorkspace(
	ROLE Admin;
	TABLE CTable INHERITS sys.CDoc (
		RefField ref(Admin)
	));
	`)
	require.NoError(err)

	pkg, err := BuildPackageSchema("test", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.Contains(err.Error(), "undefined table: Admin")

}

func Test_VRestaurantBasic(t *testing.T) {

	require := require.New(t)

	vRestaurantPkgAST, err := ParsePackageDir("github.com/untillpro/vrestaurant", fsvRestaurant, "sql_example_app/vrestaurant")
	require.NoError(err)

	packages, err := BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		vRestaurantPkgAST,
	})
	require.NoError(err)

	builder := appdef.New()
	err = BuildAppDefs(packages, builder)
	require.NoError(err)

	app, err := builder.Build()
	require.NoError(err)

	// table
	cdoc := app.Type(appdef.NewQName("vrestaurant", "TablePlan"))
	require.NotNil(cdoc)
	require.Equal(appdef.TypeKind_CDoc, cdoc.Kind())
	require.Equal(appdef.DataKind_RecordID, cdoc.(appdef.IFields).Field("Picture").DataKind())

	cdoc = app.Type(appdef.NewQName("vrestaurant", "Client"))
	require.NotNil(cdoc)

	cdoc = app.Type(appdef.NewQName("vrestaurant", "POSUser"))
	require.NotNil(cdoc)

	cdoc = app.Type(appdef.NewQName("vrestaurant", "Department"))
	require.NotNil(cdoc)

	cdoc = app.Type(appdef.NewQName("vrestaurant", "Article"))
	require.NotNil(cdoc)

	// child table
	crec := app.Type(appdef.NewQName("vrestaurant", "TableItem"))
	require.NotNil(crec)
	require.Equal(appdef.TypeKind_CRecord, crec.Kind())
	require.Equal(appdef.DataKind_int32, crec.(appdef.IFields).Field("Tableno").DataKind())

	// view
	view := appdef.View(app.Type, appdef.NewQName("vrestaurant", "SalesPerDay"))
	require.NotNil(view)
	require.Equal(appdef.TypeKind_ViewRecord, view.Kind())
}

func Test_AppSchemaErrors(t *testing.T) {
	require := require.New(t)
	fs, err := ParseFile("example2.vsql", ``)
	require.NoError(err)
	pkg2, err := BuildPackageSchema("github.com/untillpro/airsbp3/pkg2", []*FileSchemaAST{fs})
	require.NoError(err)

	fs, err = ParseFile("example3.vsql", ``)
	require.NoError(err)
	pkg3, err := BuildPackageSchema("github.com/untillpro/airsbp3/pkg3", []*FileSchemaAST{fs})
	require.NoError(err)

	f := func(sql string, expectErrors ...string) {
		ast, err := ParseFile("file2.vsql", sql)
		require.NoError(err)
		pkg, err := BuildPackageSchema("github.com/untillpro/airsbp3/pkg4", []*FileSchemaAST{ast})
		require.NoError(err)

		_, err = BuildAppSchema([]*PackageSchemaAST{
			pkg, pkg2, pkg3,
		})
		require.EqualError(err, strings.Join(expectErrors, "\n"))
	}

	f(`IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg3';
	APPLICATION test(
		USE air1;
		USE pkg3;
		)`, "file2.vsql:3:3: air1 undefined",
		"application does not define use of package github.com/untillpro/airsbp3/pkg2. Check if the package is defined in IMPORT SCHEMA and parsed under the same name")

	f(`IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg2' AS air1;
		IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg3';
		APPLICATION test(
			USE air1;
			USE pkg3;
			USE pkg3;
		)`, "file2.vsql:6:4: package with the same name already included in application")

	f(`IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg2' AS air1;
		IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg3';
		APPLICATION test(
			USE air1;
			USE pkg3;
		);
		APPLICATION test(
			USE air1;
			USE pkg3;
		)`, "file2.vsql:7:3: redefinition of application")

	f(`IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg2' AS air1;
		IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg3';
		`, "application not defined")

	f(`IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkgX' AS air1;
		IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg3';
		APPLICATION test(
			USE pkg3;
			USE air1;
		)
		`, "file2.vsql:5:4: could not import github.com/untillpro/airsbp3/pkgX. Check if the package is parsed under exactly this name")
}

func Test_AppIn2Schemas(t *testing.T) {
	require := require.New(t)
	fs, err := ParseFile("example2.vsql", `APPLICATION test1();`)
	require.NoError(err)
	pkg2, err := BuildPackageSchema("github.com/untillpro/airsbp3/pkg2", []*FileSchemaAST{fs})
	require.NoError(err)

	fs, err = ParseFile("example3.vsql", `APPLICATION test2();`)
	require.NoError(err)
	pkg3, err := BuildPackageSchema("github.com/untillpro/airsbp3/pkg3", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		pkg2, pkg3,
	})
	require.ErrorContains(err, "redefinition of application")
}

func Test_Scope_TableRefs(t *testing.T) {
	require := require.New(t)

	// *****  main
	fs, err := ParseFile("example1.vsql", `
	IMPORT SCHEMA 'github.com/untillpro/airsbp3/pkg1';
	APPLICATION test(
		USE pkg1;
	);
	`)
	require.NoError(err)
	main, err := BuildPackageSchema("github.com/untillpro/airsbp3/main", []*FileSchemaAST{fs})
	require.NoError(err)

	// *****  pkg1
	fs, err = ParseFile("example2.vsql", `
	WORKSPACE myWorkspace1 (
		TABLE PkgTable INHERITS sys.CRecord();
		TABLE MyTable INHERITS sys.CDoc (
			Items TABLE MyInnerTable()
		);
		TABLE MyTable2 INHERITS sys.CDoc (
			r1 ref(MyTable),
			r2 ref(MyTable2),
			r3 ref(PkgTable),
			r4 ref(MyInnerTable)
		);
	);
	WORKSPACE myWorkspace2 (
		TABLE MyTable3 INHERITS sys.CDoc (
			r1 ref(MyTable),
			r2 ref(MyTable2),
			r3 ref(PkgTable),
			r4 ref(MyInnerTable)
		);
	);
	`)
	require.NoError(err)
	pkg1, err := BuildPackageSchema("github.com/untillpro/airsbp3/pkg1", []*FileSchemaAST{fs})
	require.NoError(err)
	_, err = BuildAppSchema([]*PackageSchemaAST{getSysPackageAST(), main, pkg1})
	require.EqualError(err, strings.Join([]string{
		"example2.vsql:16:11: undefined table: MyTable",
		"example2.vsql:17:11: undefined table: MyTable2",
		"example2.vsql:18:11: undefined table: PkgTable",
		"example2.vsql:19:11: undefined table: MyInnerTable",
	}, "\n"))

}

func Test_Alter_Workspace_In_Package(t *testing.T) {

	require := require.New(t)

	fs0, err := ParseFile("file0.vsql", `
	IMPORT SCHEMA 'org/pkg1';
	IMPORT SCHEMA 'org/pkg2';
	APPLICATION test(
		USE pkg1;
	);
	`)
	require.NoError(err)
	pkg0, err := BuildPackageSchema("org/main", []*FileSchemaAST{fs0})
	require.NoError(err)

	fs1, err := ParseFile("file1.vsql", `
		ALTERABLE WORKSPACE Ws0(
			TABLE wst01 INHERITS sys.CDoc();
		);
		ABSTRACT WORKSPACE AWs(
			TABLE awst1 INHERITS sys.CDoc();
		);
		WORKSPACE Ws(
			TABLE wst1 INHERITS sys.CDoc();
		);
	`)
	require.NoError(err)
	fs2, err := ParseFile("file2.vsql", `
		ALTER WORKSPACE Ws0(
			TABLE wst02 INHERITS sys.CDoc();
		);
		ALTER WORKSPACE AWs(
			TABLE awst2 INHERITS sys.CDoc();
		);
		ALTER WORKSPACE Ws(
			TABLE wst2 INHERITS sys.CDoc();
		);
	`)
	require.NoError(err)
	pkg1, err := BuildPackageSchema("org/pkg1", []*FileSchemaAST{fs1, fs2})
	require.NoError(err)

	_, err = BuildAppSchema([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg0,
		pkg1,
	})
	require.NoError(err)
}

func Test_Storages(t *testing.T) {
	require := require.New(t)
	fs, err := ParseFile("example2.vsql", `APPLICATION test1();
	EXTENSION ENGINE BUILTIN (
		STORAGE MyStorage(
			INSERT SCOPE(PROJECTORS)
		);
	)
	`)
	require.NoError(err)
	pkg2, err := BuildPackageSchema("github.com/untillpro/airsbp3/pkg2", []*FileSchemaAST{fs})
	require.NoError(err)

	schema, err := BuildAppSchema([]*PackageSchemaAST{
		pkg2,
	})
	require.ErrorContains(err, "storages are only declared in sys package")
	require.Nil(schema)
}

func buildPackage(sql string) *PackageSchemaAST {
	fs, err := ParseFile("source.vsql", sql)
	if err != nil {
		panic(err)
	}
	pkg, err := BuildPackageSchema("github.com/voedger/voedger/app1", []*FileSchemaAST{fs})
	if err != nil {
		panic(err)
	}
	return pkg
}

func Test_OdocCmdArgs(t *testing.T) {
	require := require.New(t)
	pkgApp1 := buildPackage(`

	APPLICATION registry();
	WORKSPACE Workspace1 (
		TABLE TableODoc INHERITS sys.ODoc (
			orecord1 TABLE orecord1(
				orecord2 TABLE orecord2()
			)
		);
		EXTENSION ENGINE BUILTIN (
			COMMAND CmdODoc1(TableODoc) RETURNS TableODoc;
		)
	);


	`)

	schema, err := BuildAppSchema([]*PackageSchemaAST{pkgApp1, getSysPackageAST()})
	require.NoError(err)

	builder := appdef.New()
	err = BuildAppDefs(schema, builder)
	require.NoError(err)

	app, err := builder.Build()
	require.NoError(err)

	cmdOdoc := appdef.Command(app.Type, appdef.NewQName("app1", "CmdODoc1"))
	require.NotNil(cmdOdoc)
	require.NotNil(cmdOdoc.Param())

	odoc := cmdOdoc.Param().(appdef.IContainers)
	require.Equal(1, odoc.ContainerCount())
	require.Equal("orecord1", odoc.Container("orecord1").Name())
	container := odoc.Container("orecord1")
	require.Equal(appdef.Occurs(0), container.MinOccurs())
	require.Equal(appdef.Occurs(100), container.MaxOccurs())

	orec := appdef.ORecord(app.Type, appdef.NewQName("app1", "orecord1"))
	require.NotNil(orec)
	require.Equal(1, orec.ContainerCount())
	require.Equal("orecord2", orec.Container("orecord2").Name())

}

func Test_TypeContainers(t *testing.T) {
	require := require.New(t)
	pkgApp1 := buildPackage(`

APPLICATION registry();

WORKSPACE Workspace1 (
	TYPE Person (
		Name 	varchar,
		Age 	int32
	);

	TYPE Item (
		Name 	varchar,
		Price 	currency
	);

	TYPE Deal (
		side1 		Person NOT NULL,	-- collection 1..1
		side2 		Person				-- collection 0..1
	--	items 		Item[] NOT NULL,	-- (not yet supported by kernel) collection 1..* (up to maxNestedTableContainerOccurrences = 100)
	--	discounts 	Item[3]				-- (not yet supported by kernel) collection 0..3 (one-based numbering convention for arrays, similarly to PostgreSQL)
	);
	EXTENSION ENGINE BUILTIN (
		COMMAND CmdDeal(Deal) RETURNS Deal;
	)
);
	`)

	schema, err := BuildAppSchema([]*PackageSchemaAST{pkgApp1, getSysPackageAST()})
	require.NoError(err)

	builder := appdef.New()
	err = BuildAppDefs(schema, builder)
	require.NoError(err)

	validate := func(par appdef.IType) {
		o, ok := par.(appdef.IObject)
		require.True(ok, "expected %v supports IObject", par)
		require.Equal(2, o.ContainerCount())
		require.Equal(appdef.Occurs(1), o.Container("side1").MinOccurs())
		require.Equal(appdef.Occurs(1), o.Container("side1").MaxOccurs())
		require.Equal(appdef.Occurs(0), o.Container("side2").MinOccurs())
		require.Equal(appdef.Occurs(1), o.Container("side2").MaxOccurs())

		/* TODO: uncomment when kernel supports it
		require.Equal(appdef.Occurs(1), o.Container("items").MinOccurs())
		require.Equal(appdef.Occurs(100), o.Container("items").MaxOccurs())
		require.Equal(appdef.Occurs(0), o.Container("discounts").MinOccurs())
		require.Equal(appdef.Occurs(3), o.Container("discounts").MaxOccurs())
		*/
	}

	app, err := builder.Build()
	require.NoError(err)

	cmd := appdef.Command(app.Type, appdef.NewQName("app1", "CmdDeal"))
	validate(cmd.Param())
	validate(cmd.Result())
}

func Test_EmptyType(t *testing.T) {
	require := require.New(t)
	pkgApp1 := buildPackage(`

APPLICATION registry(); WORKSPACE Workspace1 (
	TYPE EmptyType ();
)
	`)

	schema, err := BuildAppSchema([]*PackageSchemaAST{pkgApp1, getSysPackageAST()})
	require.NoError(err)

	builder := appdef.New()
	err = BuildAppDefs(schema, builder)
	require.NoError(err)

	app, err := builder.Build()
	require.NoError(err)

	cdoc := appdef.Object(app.Type, appdef.NewQName("app1", "EmptyType"))
	require.NotNil(cdoc)
}

func Test_EmptyType1(t *testing.T) {
	require := require.New(t)
	pkgApp1 := buildPackage(`

APPLICATION registry(); WORKSPACE Workspace1 (


TYPE SomeType (
	t int321
);

TABLE SomeTable INHERITS sys.CDoc (
	t int321
))
	`)

	_, err := BuildAppSchema([]*PackageSchemaAST{pkgApp1, getSysPackageAST()})
	require.EqualError(err, strings.Join([]string{
		"source.vsql:7:4: undefined type: int321",
		"source.vsql:11:4: undefined data type or table: int321",
	}, "\n"))

}

func Test_ODocUnknown(t *testing.T) {
	require := require.New(t)
	pkgApp1 := buildPackage(`APPLICATION registry(); WORKSPACE Workspace1 (
TABLE MyTable1 INHERITS ODocUnknown ( MyField ref(registry.Login) NOT NULL ));
`)

	_, err := BuildAppSchema([]*PackageSchemaAST{pkgApp1, getSysPackageAST()})
	require.EqualError(err, strings.Join([]string{
		"source.vsql:2:1: undefined table kind",
	}, "\n"))

}

//go:embed package.vsql
var pkgSqlFS embed.FS

func TestParseFilesFromFSRoot(t *testing.T) {
	t.Run("dot", func(t *testing.T) {
		_, err := ParsePackageDir("github.com/untillpro/main", pkgSqlFS, ".")
		require.NoError(t, err)
	})
}

func Test_Constraints(t *testing.T) {
	require := assertions(t)

	require.AppSchemaError(`
	APPLICATION app1(); WORKSPACE ws1 (
	TABLE SomeTable INHERITS sys.CDoc (
		t1 int32,
		t2 int32,
		CONSTRAINT c1 UNIQUE(t1),
		CONSTRAINT c1 UNIQUE(t2)
	))`, "file.vsql:7:3: redefinition of c1")

	require.AppSchemaError(`
	APPLICATION app1(); WORKSPACE ws1 (
	TABLE SomeTable INHERITS sys.CDoc (
		UNIQUEFIELD UnknownField
	))`, "file.vsql:4:3: undefined field UnknownField")

	require.AppSchemaError(`
	APPLICATION app1(); WORKSPACE ws1 (
	TABLE SomeTable INHERITS sys.CDoc (
		t1 int32,
		t2 int32,
		CONSTRAINT c1 UNIQUE(t1),
		CONSTRAINT c2 UNIQUE(t2, t1)
	))`, "file.vsql:7:3: field t1 already in unique constraint")

}

func Test_Grants(t *testing.T) {
	require := assertions(t)

	t.Run("Basic", func(t *testing.T) {
		require.AppSchemaError(`
	APPLICATION app1();
	WORKSPACE ws1 (
		ROLE role1;
		GRANT ALL ON TABLE Fake TO app1;
		GRANT EXECUTE ON COMMAND Fake TO role1;
		GRANT EXECUTE ON QUERY Fake TO role1;
		TABLE Tbl INHERITS sys.CDoc();
		GRANT ALL(FakeCol) ON TABLE Tbl TO role1;
		GRANT INSERT,UPDATE(FakeCol) ON TABLE Tbl TO role1;
		GRANT EXECUTE ON ALL COMMANDS WITH TAG x TO role1;
		TABLE Nested1 INHERITS sys.CRecord();
		TABLE Tbl2 INHERITS sys.CDoc(
			ref1 ref(Tbl),
			items TABLE Nested(),
			items2 Nested1
		);
		GRANT ALL(ref1) ON TABLE Tbl2 TO role1;
		GRANT ALL(items) ON TABLE Tbl2 TO role1;
		GRANT ALL(items2) ON TABLE Tbl2 TO role1;
		GRANT SELECT ON VIEW Fake TO role1;
		GRANT SELECT ON ALL VIEWS WITH TAG x TO role1;
	);
	`, "file.vsql:5:30: undefined role: app1",
			"file.vsql:5:22: Fake undefined",
			"file.vsql:6:28: undefined command: Fake",
			"file.vsql:7:26: undefined query: Fake",
			"file.vsql:9:13: undefined field FakeCol",
			"file.vsql:10:23: undefined field FakeCol",
			"file.vsql:11:42: undefined tag: x",
			"file.vsql:21:24: undefined view: Fake",
			"file.vsql:22:38: undefined tag: x",
		)
	})

	t.Run("GRANT follows REVOKE in WORKSPACE", func(t *testing.T) {
		require.AppSchemaError(`APPLICATION test();
			WORKSPACE AppWorkspaceWS (
				ROLE role1;

				TABLE Table1 INHERITS sys.CDoc(
					Field1 int32
				);
				REVOKE ALL ON TABLE Table1 FROM role1;
				GRANT ALL ON TABLE Table1 TO role1;

			);`, "file.vsql:9:5: GRANT follows REVOKE in the same container")
	})

	t.Run("GRANT Role", func(t *testing.T) {
		schema, err := require.AppSchema(`APPLICATION test();
			ABSTRACT WORKSPACE BaseWs (
				ROLE admin;
			);
			WORKSPACE Workspace1 INHERITS BaseWs (
				ROLE mgr;
				GRANT admin TO mgr;
			);
		`)
		require.NoError(err)
		builder := appdef.New()
		err = BuildAppDefs(schema, builder)
		require.NoError(err)

		app, err := builder.Build()
		require.NoError(err)
		var numACLs int

		// table
		app.ACL(func(i appdef.IACLRule) bool {
			require.Len(i.Ops(), 1)
			require.Equal(appdef.OperationKind_Inherits, i.Ops()[0])
			require.Equal(appdef.PolicyKind_Allow, i.Policy())
			require.Len(i.Resources().On(), 1)
			require.Equal("pkg.admin", i.Resources().On()[0].String())
			require.Equal("pkg.mgr", i.Principal().QName().String())
			numACLs++
			return true
		})
		require.Equal(1, numACLs)
	})

	t.Run("GRANT to descriptor", func(t *testing.T) {
		schema, err := require.AppSchema(`APPLICATION test();
			ALTERABLE WORKSPACE UserProfileWS (
				DESCRIPTOR UserProfile (
					DisplayName varchar
				);
				ROLE ProfileOwner;
        		GRANT SELECT ON TABLE UserProfile TO ProfileOwner;
			);
		`)
		require.NoError(err)
		builder := appdef.New()
		err = BuildAppDefs(schema, builder)
		require.NoError(err)

		app, err := builder.Build()
		require.NoError(err)
		var numACLs int

		// table
		app.ACL(func(i appdef.IACLRule) bool {
			require.Len(i.Ops(), 1)
			require.Equal(appdef.OperationKind_Select, i.Ops()[0])
			require.Equal(appdef.PolicyKind_Allow, i.Policy())
			require.Len(i.Resources().On(), 1)
			require.Equal("pkg.UserProfile", i.Resources().On()[0].String())
			require.Equal("pkg.ProfileOwner", i.Principal().QName().String())
			numACLs++
			return true
		})
		require.Equal(1, numACLs)
	})

}

func Test_Grants_Inherit(t *testing.T) {
	require := assertions(t)

	t.Run("GRANT ALL does not include resources from inherited workspaces", func(t *testing.T) {
		schema, err := require.AppSchema(`APPLICATION test();
			ABSTRACT WORKSPACE BaseWs (
				ROLE role1;
				TABLE Table1 INHERITS sys.CDoc();
			);
			WORKSPACE AppWorkspaceWS INHERITS BaseWs (
				DESCRIPTOR AppWorkspace();
				TABLE Table2 INHERITS sys.CDoc();
				GRANT INSERT ON ALL TABLES TO role1;
			);`)
		require.NoError(err)
		builder := appdef.New()
		err = BuildAppDefs(schema, builder)
		require.NoError(err)

		app, err := builder.Build()
		require.NoError(err)
		var numACLs int

		// table
		app.ACL(func(i appdef.IACLRule) bool {
			require.Len(i.Ops(), 1)
			require.Equal(appdef.OperationKind_Insert, i.Ops()[0])
			require.Equal(appdef.PolicyKind_Allow, i.Policy())
			require.Len(i.Resources().On(), 2)
			require.True(i.Resources().On().ContainsAll(appdef.NewQName("pkg", "Table2"), appdef.NewQName("pkg", "AppWorkspace")))
			require.Equal("pkg.role1", i.Principal().QName().String())
			numACLs++
			return true
		})
		require.Equal(1, numACLs)
	})

	t.Run("GRANT ALL * WITH TAG includes resources from inherited workspaces", func(t *testing.T) {
		schema, err := require.AppSchema(`APPLICATION test();
			TAG tag1;
			ABSTRACT WORKSPACE BaseWs (
				ROLE role1;
				TABLE Table1 INHERITS sys.CDoc() WITH Tags=(tag1);
			);
			WORKSPACE AppWorkspaceWS INHERITS BaseWs (
				TABLE Table2 INHERITS sys.CDoc() WITH Tags=(tag1);
				GRANT INSERT ON ALL TABLES WITH TAG tag1 TO role1;
			);`)
		require.NoError(err)
		builder := appdef.New()
		err = BuildAppDefs(schema, builder)
		require.NoError(err)

		app, err := builder.Build()
		require.NoError(err)
		var numACLs int

		// table
		app.ACL(func(i appdef.IACLRule) bool {
			require.Len(i.Ops(), 1)
			require.Equal(appdef.OperationKind_Insert, i.Ops()[0])
			require.Equal(appdef.PolicyKind_Allow, i.Policy())
			require.Len(i.Resources().On(), 2)
			require.Equal("pkg.Table1", i.Resources().On()[0].String())
			require.Equal("pkg.Table2", i.Resources().On()[1].String())
			require.Equal("pkg.role1", i.Principal().QName().String())
			numACLs++
			return true
		})
		require.Equal(1, numACLs)
	})

}

func Test_UndefinedType(t *testing.T) {
	require := assertions(t)

	require.AppSchemaError(`APPLICATION app1(); WORKSPACE w (
TABLE MyTable2 INHERITS sys.ODoc (
MyField int23 NOT NULL
))
	`, "file.vsql:3:9: undefined data type or table: int23",
	)
}

func Test_DescriptorInProjector(t *testing.T) {
	require := assertions(t)

	require.AppSchemaError(`APPLICATION app1();
	WORKSPACE w (
		EXTENSION ENGINE BUILTIN (
		  PROJECTOR x AFTER INSERT ON (unknown.z) STATE(sys.Http);
		);
	  );
	`,
		"file.vsql:4:34: unknown undefined")

	require.NoAppSchemaError(`APPLICATION app1();
	WORKSPACE RestaurantWS (
		DESCRIPTOR Restaurant ();
		EXTENSION ENGINE BUILTIN (
		  PROJECTOR NewRestaurantVat AFTER INSERT OR UPDATE ON (Restaurant) STATE(sys.AppSecret, sys.Http) INTENTS(sys.SendMail);
		);
	  );
	`)
}

type testVarResolver struct {
	resolved map[appdef.QName]bool
}

func (t testVarResolver) AsInt32(name appdef.QName) (int32, bool) {
	t.resolved[name] = true
	return 1, true
}

func Test_Variables(t *testing.T) {
	require := assertions(t)

	require.AppSchemaError(`APPLICATION app1(); RATE AppDefaultRate variable PER HOUR;`, "file.vsql:1:41: variable undefined")

	schema, err := require.AppSchema(`APPLICATION app1();
	DECLARE variable int32 DEFAULT 100;
	RATE AppDefaultRate variable PER HOUR;
	`)
	require.NoError(err)

	resolver := testVarResolver{resolved: make(map[appdef.QName]bool)}

	BuildAppDefs(schema, appdef.New(), WithVariableResolver(&resolver))
	require.True(resolver.resolved[appdef.NewQName("pkg", "variable")])
}

func Test_RatesAndLimits(t *testing.T) {
	require := assertions(t)

	require.AppSchemaError(`APPLICATION app1();
	WORKSPACE w (
		RATE r 1 PER HOUR;
		LIMIT l1 ON EVERYTHING WITH RATE x;
		LIMIT l2 ON COMMAND x WITH RATE r;
		LIMIT l3 ON QUERY y WITH RATE r;
		LIMIT l4 ON TAG z WITH RATE r;
		LIMIT l5 ON TABLE t WITH RATE r;
	);`,
		"file.vsql:4:36: undefined rate: x",
		"file.vsql:5:23: undefined command: x",
		"file.vsql:6:21: undefined query: y",
		"file.vsql:7:19: undefined tag: z",
		"file.vsql:8:21: undefined table: t")
}

func Test_RefsFromInheritedWs(t *testing.T) {
	require := assertions(t)

	require.NoAppSchemaError(`APPLICATION test();
	ABSTRACT WORKSPACE base (
		TABLE tab1 INHERITS sys.WDoc (
			Fld1 int32
		);
	);
	WORKSPACE work INHERITS base (
		TABLE tab2 INHERITS sys.WDoc (
			Fld2 ref(tab1)
		);
	);`)
}

//go:embed test/pkg1.vsql
var pkg1FS embed.FS

func Test_Panic1(t *testing.T) {
	ast, errs := ParsePackageDir(appdef.SysPackage, pkg1FS, "test")
	require.ErrorContains(t, errs, "no valid schema files")
	require.Nil(t, ast)
}

func Test_Identifiers(t *testing.T) {
	require := assertions(t)

	_, err := ParseFile("file1.vsql", `APPLICATION app1();
	WORKSPACE w (
		ROLE _role;
	);`)
	require.ErrorContains(err, "file1.vsql:3:8: invalid input text")

	_, err = ParseFile("file1.vsql", `APPLICATION app1();
	WORKSPACE w (
		ROLE r234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890r23456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456;
	);`)
	require.ErrorContains(err, "file1.vsql:3:263: unexpected token")

	_, err = ParseFile("file1.vsql", `APPLICATION app1();
	WORKSPACE w (
		ROLE r世界;
	);`)
	require.ErrorContains(err, "file1.vsql:3:9: invalid input text")
}

func Test_RefsWorkspaces(t *testing.T) {
	require := assertions(t)

	require.NoAppSchemaError(`APPLICATION test();
	WORKSPACE w2 (
		TABLE t1 INHERITS sys.WDoc(
			items TABLE t2(
				items TABLE t3()
			)
		);
		TABLE tab2 INHERITS sys.WDoc(
			f1 ref(t2),
			f2 ref(t3)
		);
		TYPE typ2(
			f1 ref(t2),
			f2 ref(t3)
		);
		VIEW test(
			f1 ref(t2),
			f2 ref(t3),
			PRIMARY KEY(f1)
		) AS RESULT OF Proj1;

		EXTENSION ENGINE BUILTIN (
			PROJECTOR Proj1 AFTER EXECUTE ON (Orders) INTENTS (sys.View(test));
			COMMAND Orders()
		);

	);`)
}

func Test_ScheduledProjectors(t *testing.T) {

	t.Run("should be deprecated", func(t *testing.T) {
		require := assertions(t)
		require.AppSchemaError(
			`APPLICATION test();
				ALTER WORKSPACE sys.AppWorkspaceWS (
					EXTENSION ENGINE BUILTIN (
						PROJECTOR ScheduledProjector CRON '1 0 * * *';
					);
				);`,
			"file.vsql:4:7: scheduled projector deprecated; use jobs instead")
	})

	t.Run("bad workspace", func(t *testing.T) {
		t.Skip()
		require := assertions(t)
		require.AppSchemaError(`APPLICATION test();
			WORKSPACE w2 (
				EXTENSION ENGINE BUILTIN (
					PROJECTOR Proj1 CRON '1 0 * * *';
				);
			);`, "file.vsql:4:6: scheduled projector must be in app workspace")
	})

	t.Run("bad cron and intents", func(t *testing.T) {
		t.Skip()
		require := assertions(t)
		require.AppSchemaError(`APPLICATION test();
			ALTER WORKSPACE AppWorkspaceWS (
				VIEW test(
					i int32,
					PRIMARY KEY(i)
				) AS RESULT OF Proj1;

				EXTENSION ENGINE BUILTIN (
					PROJECTOR Proj1 CRON 'blah' INTENTS (sys.View(test));
				);
			);`, "file.vsql:9:6: invalid cron schedule: blah", "file.vsql:9:6: scheduled projector cannot have intents")
	})

	t.Run("good cron", func(t *testing.T) {
		t.Skip()
		require := assertions(t)
		require.NoAppSchemaError(`APPLICATION test();
ALTER WORKSPACE sys.AppWorkspaceWS (
	EXTENSION ENGINE BUILTIN (
		PROJECTOR ScheduledProjector CRON '1 0 * * *';
	);
);`)
	})
}

func Test_Jobs(t *testing.T) {

	t.Run("bad workspace", func(t *testing.T) {
		t.Skip()
		require := assertions(t)
		require.AppSchemaError(`APPLICATION test();
			WORKSPACE w2 (
				EXTENSION ENGINE BUILTIN (
					JOB Job1 '1 0 * * *';
				);
			);`, "file.vsql:4:6: job must be in app workspace")
	})

	t.Run("bad cron", func(t *testing.T) {
		require := assertions(t)
		require.AppSchemaError(`APPLICATION test();
			ALTER WORKSPACE sys.AppWorkspaceWS (
				EXTENSION ENGINE BUILTIN (
					JOB Job1 'blah';
				);
			);`, "file.vsql:4:6: invalid cron schedule: blah")
	})

	t.Run("good cron", func(t *testing.T) {
		require := assertions(t)
		require.NoAppSchemaError(`APPLICATION test();
			ALTER WORKSPACE sys.AppWorkspaceWS (
				EXTENSION ENGINE BUILTIN (
					JOB Job1 '1 0 * * *';
				);
			);`)
	})
}

func Test_DataTypes(t *testing.T) {

	require := assertions(t)
	require.NoAppSchemaError(`APPLICATION test();
ALTER WORKSPACE sys.AppWorkspaceWS (
	TABLE t1 INHERITS sys.WDoc(
		s1_1_1 character varying(10),
		s1_1_1 character varying,
		s1_2_1 varchar(10),
		s1_2_2 varchar,
		s1_3_1 text(10),
		s1_3_1 text,

		s2_1_1 binary varying(10),
		s2_1_1 binary varying,
		s2_2_1 varbinary(10),
		s2_2_2 varbinary,
		s2_3_1 bytes(10),
		s2_3_1 bytes,

		s3_1 bigint,
		s3_2 int64,

		s4_1 integer,
		s4_2 int32,
		s4_3 int,

		s5_1 real,
		s5_2 float,
		s5_3 float32,

		s6_1 double precision,
		s6_2 float64,

		s7_2 money,
		s7_3 currency,

		s8_1 boolean,
		s8_2 bool,

		s9_1 binary large object,
		s9_2 blob,

		s10_1 qualified name,
		s10_2 qname

	);
);`)
}

func Test_UniquesFromFieldsets(t *testing.T) {
	require := assertions(t)
	schema, err := require.AppSchema(`APPLICATION test(); WORKSPACE w (
	TYPE fieldset (
		f1 int32
	);
	TABLE t1 INHERITS sys.WDoc(
		fieldset,
		f2 int32,
		UNIQUE(f1)
	);
)`)
	require.NoError(err)
	require.NoError(BuildAppDefs(schema, appdef.New()))
}

func Test_CRecordInDescriptor(t *testing.T) {
	require := assertions(t)
	schema, err := require.AppSchema(`APPLICATION test();
	WORKSPACE w (
		DESCRIPTOR wd(
			items x
		);
		TABLE x INHERITS sys.CRecord(
			f1 int32
		);
	);
`)
	require.NoError(err)
	require.NoError(BuildAppDefs(schema, appdef.New()))
}

func Test_RefInheritedFromSys(t *testing.T) {
	require := assertions(t)

	_, err := require.AppSchema(`APPLICATION SomeApp();
	WORKSPACE SomeWS (
	    TABLE SomeTable INHERITS sys.CDoc(
	        ChildWorkspaceID ref(sys.ChildWorkspace)
    	);
	)
	`)
	require.NoError(err)
}
