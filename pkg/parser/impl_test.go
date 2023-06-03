/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

import (
	"embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/voedger/voedger/pkg/appdef"
)

//go:embed example_app/*.sql
var efs embed.FS

//go:embed system_pkg/*.sql
var sfs embed.FS

//_go:embed example_app/expectedParsed.schema
//var expectedParsedExampledSchemaStr string

func getSysPackageAST() *PackageSchemaAST {
	pkgSys, err := ParsePackageDir(appdef.SysPackage, sfs, "system_pkg")
	if err != nil {
		panic(err)
	}
	return pkgSys
}

func Test_BasicUsage(t *testing.T) {

	require := require.New(t)
	examplePkgAST, err := ParsePackageDir("github.com/untillpro/exampleschema", efs, "example_app")
	require.NoError(err)

	// := repr.String(pkgExample, repr.Indent(" "), repr.IgnorePrivate())
	//fmt.Println(parsedSchemaStr)

	packages, err := MergePackageSchemas([]*PackageSchemaAST{
		getSysPackageAST(),
		examplePkgAST,
	})
	require.NoError(err)

	builder := appdef.New()
	err = BuildAppDefs(packages, builder)
	require.NoError(err)

	// table
	cdoc := builder.Def(appdef.NewQName("air", "AirTablePlan"))
	require.NotNil(cdoc)
	require.Equal(appdef.DefKind_CDoc, cdoc.Kind())
	require.Equal(appdef.DataKind_int32, cdoc.(appdef.IFields).Field("FState").DataKind())

	// child table
	crec := builder.Def(appdef.NewQName("air", "AirTablePlanItem"))
	require.NotNil(crec)
	require.Equal(appdef.DefKind_CRecord, crec.Kind())
	require.Equal(appdef.DataKind_int32, crec.(appdef.IFields).Field("TableNo").DataKind())

	// type
	obj := builder.Object(appdef.NewQName("air", "SubscriptionEvent"))
	require.Equal(appdef.DefKind_Object, obj.Kind())
	require.Equal(appdef.DataKind_string, obj.Field("Origin").DataKind())

	// view
	view := builder.View(appdef.NewQName("air", "XZReports"))
	require.NotNil(view)
	require.Equal(appdef.DefKind_ViewRecord, view.Kind())

	require.Equal(1, view.Value().UserFieldCount())
	require.Equal(1, view.Key().PartKey().FieldCount())
	require.Equal(4, view.Key().ClustCols().FieldCount())
}

func Test_DupFieldsInTypes(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("file1.sql", `SCHEMA test;
	TYPE RootType (
		Id int32
	);
	TYPE BaseType OF RootType(
		baseField int
	);
	TYPE BaseType2 (
		someField int
	);
	TYPE MyType OF BaseType, BaseType2 (
		field text,
		field text,
		baseField text,
		someField int,
		Id text
	)
	`)
	require.NoError(err)
	pkg, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.NoError(err)

	packages, err := MergePackageSchemas([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	err = BuildAppDefs(packages, appdef.New())
	require.EqualError(err, strings.Join([]string{
		"file1.sql:13:3: field redeclared",
		"file1.sql:14:3: baseField redeclared",
		"file1.sql:15:3: someField redeclared",
		"file1.sql:16:3: Id redeclared",
	}, "\n"))

}

func Test_DupFieldsInTables(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("file1.sql", `SCHEMA test;
	TYPE RootType (
		Kind int32
	);
	TYPE BaseType OF RootType(
		baseField int
	);
	TYPE BaseType2 (
		someField int
	);
	TABLE ByBaseTable INHERITS CDoc (
		Name text,
		Code text
	);
	TABLE MyTable INHERITS ByBaseTable OF BaseType, BaseType2 (
		newField text,
		field text,
		field text, 		-- duplicated in the this table
		baseField text,		-- duplicated in the first OF
		someField int,		-- duplicated in the second OF
		Kind int,			-- duplicated in the first OF (2nd level)
		Name int,			-- duplicated in the inherited table
		ID text				
	)
	`)
	require.NoError(err)
	pkg, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.NoError(err)

	packages, err := MergePackageSchemas([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	err = BuildAppDefs(packages, appdef.New())
	require.EqualError(err, strings.Join([]string{
		"file1.sql:18:3: field redeclared",
		"file1.sql:19:3: baseField redeclared",
		"file1.sql:20:3: someField redeclared",
		"file1.sql:21:3: Kind redeclared",
		"file1.sql:22:3: Name redeclared",
	}, "\n"))

}

func Test_Expressions(t *testing.T) {
	require := require.New(t)

	_, err := ParseFile("file1.sql", `SCHEMA test; 
	TABLE MyTable(
		Int1 text DEFAULT 1 CHECK(Int1 > Int2),
		Int1 int DEFAULT 1 CHECK(Text != "asd"),
		Int1 int DEFAULT 1 CHECK(Int2 > -5),
		Int1 int DEFAULT 1 CHECK(TextField > "asd" AND (SomeFloat/3.2)*4 != 5.003),
		Int1 int DEFAULT 1 CHECK(SomeFunc("a", TextField) AND BoolField=FALSE),
		
		CHECK(MyRowValidator(this))
	)
	`)
	require.NoError(err)

}

func Test_Duplicates(t *testing.T) {
	require := require.New(t)

	ast1, err := ParseFile("file1.sql", `SCHEMA test; 
	EXTENSION ENGINE BUILTIN (
		FUNCTION MyTableValidator() RETURNS void;
		FUNCTION MyTableValidator(TableRow) RETURNS string;	
		FUNCTION MyFunc2() RETURNS void;
	);
	TABLE Rec1 INHERITS CRecord();
	`)
	require.NoError(err)

	ast2, err := ParseFile("file2.sql", `SCHEMA test; 
	WORKSPACE ChildWorkspace (
		TAG MyFunc2; -- redeclared
		EXTENSION ENGINE BUILTIN (
			FUNCTION MyFunc3() RETURNS void;
			FUNCTION MyFunc4() RETURNS void;
		);
		WORKSPACE InnerWorkspace (
			ROLE MyFunc4; -- redeclared
		);
		TABLE Doc1 INHERITS ODoc(
			nested1 Rec1,
			nested2 TABLE Rec1() -- redeclared
		)
	)
	`)
	require.NoError(err)

	_, err = MergeFileSchemaASTs("", []*FileSchemaAST{ast1, ast2})

	require.EqualError(err, strings.Join([]string{
		"file1.sql:4:3: MyTableValidator redeclared",
		"file2.sql:3:3: MyFunc2 redeclared",
		"file2.sql:9:4: MyFunc4 redeclared",
		"file2.sql:13:12: Rec1 redeclared",
	}, "\n"))

}

func Test_DuplicatesInViews(t *testing.T) {
	require := require.New(t)

	ast, err := ParseFile("file2.sql", `SCHEMA test; 
	WORKSPACE Workspace (
		VIEW test(
			field1 int,
			field2 int,
			field1 text,
			PRIMARY KEY(field1),
			PRIMARY KEY(field2)			
		) AS RESULT OF Proj1;
	)
	`)
	require.NoError(err)

	_, err = MergeFileSchemaASTs("", []*FileSchemaAST{ast})

	require.EqualError(err, strings.Join([]string{
		"file2.sql:6:4: field1 redeclared",
		"file2.sql:8:4: primary key redeclared",
	}, "\n"))

}
func Test_Comments(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	EXTENSION ENGINE BUILTIN (
		-- My function
		-- line 2
		FUNCTION MyFunc() RETURNS void;
	);
	`)
	require.Nil(err)

	ps, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.Nil(err)

	require.NotNil(ps.Ast.Statements[0].ExtEngine.Statements[0].Function.Comments)
	require.Equal(2, len(ps.Ast.Statements[0].ExtEngine.Statements[0].Function.Comments))
	require.Equal("My function", ps.Ast.Statements[0].ExtEngine.Statements[0].Function.Comments[0])
	require.Equal("line 2", ps.Ast.Statements[0].ExtEngine.Statements[0].Function.Comments[1])
}

func Test_UnexpectedSchema(t *testing.T) {
	require := require.New(t)

	ast1, err := ParseFile("file1.sql", `SCHEMA schema1; ROLE abc;`)
	require.NoError(err)

	ast2, err := ParseFile("file2.sql", `SCHEMA schema2; ROLE xyz;`)
	require.NoError(err)

	_, err = MergeFileSchemaASTs("", []*FileSchemaAST{ast1, ast2})
	require.EqualError(err, "file2.sql: package schema2; expected schema1")
}

func Test_Undefined(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	WORKSPACE test (
		EXTENSION ENGINE WASM (
			COMMAND Orders() WITH Tags=[UndefinedTag];
			QUERY Query1 RETURNS text WITH Rate=UndefinedRate, Comment=xyz.UndefinedComment;
			PROJECTOR ImProjector ON COMMAND xyz.CreateUPProfile USES sys.HTTPStorage;
		)
	)
	`)
	require.Nil(err)

	pkg, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.Nil(err)

	_, err = MergePackageSchemas([]*PackageSchemaAST{pkg, getSysPackageAST()})

	require.EqualError(err, strings.Join([]string{
		"example.sql:4:4: UndefinedTag undefined",
		"example.sql:5:4: UndefinedRate undefined",
		"example.sql:5:4: xyz undefined",
		"example.sql:6:4: xyz undefined",
	}, "\n"))
}

func Test_Imports(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA pkg1;
	IMPORT SCHEMA "github.com/untillpro/airsbp3/pkg2";
	IMPORT SCHEMA "github.com/untillpro/airsbp3/pkg3" AS air;
	WORKSPACE test (
		EXTENSION ENGINE WASM (
    		COMMAND Orders WITH Tags=[pkg2.SomeTag];
    		QUERY Query1 RETURNS text WITH Comment=pkg2.SomeComment;
    		QUERY Query2 RETURNS text WITH Comment=air.SomeComment;
    		QUERY Query3 RETURNS text WITH Comment=air.SomeComment2; -- air.SomeComment2 undefined
    		PROJECTOR ImProjector ON COMMAND Air.CreateUPProfil USES sys.HTTPStorage; -- Air undefined
		)
	)
	`)
	require.NoError(err)
	pkg1, err := MergeFileSchemaASTs("github.com/untillpro/airsbp3/pkg1", []*FileSchemaAST{fs})
	require.NoError(err)

	fs, err = ParseFile("example.sql", `SCHEMA pkg2;
	TAG SomeTag;
	COMMENT SomeComment "Hello world!";
	`)
	require.NoError(err)
	pkg2, err := MergeFileSchemaASTs("github.com/untillpro/airsbp3/pkg2", []*FileSchemaAST{fs})
	require.NoError(err)

	fs, err = ParseFile("example.sql", `SCHEMA pkg3;
	COMMENT SomeComment "Hello world!";
	`)
	require.NoError(err)
	pkg3, err := MergeFileSchemaASTs("github.com/untillpro/airsbp3/pkg3", []*FileSchemaAST{fs})
	require.NoError(err)

	_, err = MergePackageSchemas([]*PackageSchemaAST{getSysPackageAST(), pkg1, pkg2, pkg3})
	require.EqualError(err, strings.Join([]string{
		"example.sql:9:7: air.SomeComment2 undefined",
		"example.sql:10:7: Air undefined",
	}, "\n"))

}

func Test_AbstractWorkspace(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	WORKSPACE ws1 ();
	ABSTRACT WORKSPACE ws2();
	ABSTRACT WORKSPACE ws3();
	WORKSPACE ws4 OF ws2,test.ws3 ();
	`)
	require.Nil(err)

	ps, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.Nil(err)

	require.False(ps.Ast.Statements[0].Workspace.Abstract)
	require.True(ps.Ast.Statements[1].Workspace.Abstract)
	require.True(ps.Ast.Statements[2].Workspace.Abstract)
	require.False(ps.Ast.Statements[3].Workspace.Abstract)
	require.Equal("ws2", ps.Ast.Statements[3].Workspace.Of[0].String())
	require.Equal("test.ws3", ps.Ast.Statements[3].Workspace.Of[1].String())

}

func Test_UniqueFields(t *testing.T) {
	require := require.New(t)

	fs, err := ParseFile("example.sql", `SCHEMA test; 
	TABLE MyTable INHERITS CDoc (
		Int1 int32,
		Int2 int32 NOT NULL,
		UNIQUEFIELD UnknownField,
		UNIQUEFIELD Int1,
		UNIQUEFIELD Int2
	)
	`)
	require.Nil(err)

	pkg, err := MergeFileSchemaASTs("", []*FileSchemaAST{fs})
	require.Nil(err)

	packages, err := MergePackageSchemas([]*PackageSchemaAST{
		getSysPackageAST(),
		pkg,
	})
	require.NoError(err)

	def := appdef.New()
	err = BuildAppDefs(packages, def)
	require.EqualError(err, strings.Join([]string{
		"example.sql:5:3: undefined field UnknownField",
		"example.sql:6:3: field has to be NOT NULL",
	}, "\n"))

	cdoc := def.CDoc(appdef.NewQName("test", "MyTable"))
	require.NotNil(cdoc)

	fld := cdoc.UniqueField()
	require.NotNil(fld)
	require.Equal("Int2", fld.Name())
}
