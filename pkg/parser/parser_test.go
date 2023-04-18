/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

import (
	"path/filepath"
	"testing"

	participle "github.com/alecthomas/participle/v2"
	"github.com/stretchr/testify/require"
)

func Test_BasicUsage(t *testing.T) {
	require := require.New(t)

	path, err := filepath.Abs("./_testdata/app1")
	require.NoError(err)
	schemas, err := ParseDir(path)
	require.NoError(err)

	require.Equal(2, len(schemas))
}

func Test_Basic(t *testing.T) {
	require := require.New(t)
	var schema = &schemaAST{}

	schema, err := ParseString2(`SCHEMA test;`)
	require.Nil(err)
	require.Equal("test", schema.Package)

	schema, err = ParseString2(`SCHEMA test; 
	IMPORT SCHEMA "github.com/untillpro/untill";
	IMPORT SCHEMA "github.com/untillpro/airsbp" AS air;		
	`)
	require.Nil(err)
	require.Equal("test", schema.Package)
	require.Equal(2, len(schema.Imports))

	require.Equal("\"github.com/untillpro/untill\"", schema.Imports[0].Name)
	require.Equal((*string)(nil), schema.Imports[0].Alias)
	require.Equal("\"github.com/untillpro/airsbp\"", schema.Imports[1].Name)
	require.Equal("air", *schema.Imports[1].Alias)

}

func Test_RootStatements(t *testing.T) {
	require := require.New(t)
	var schema = &schemaAST{}

	schema, err := ParseString2(`
	SCHEMA test; 
	TEMPLATE demo OF WORKSPACE air.Restaurant SOURCE wsTemplate_demo;
	FUNCTION MyTableValidator(TableRow) RETURNS void ENGINE WASM; 
	`)
	require.Nil(err)
	require.Equal("test", schema.Package)
	require.Equal(2, len(schema.Statements))

	require.NotNil(schema.Statements[0].Template)
	require.Equal("demo", schema.Statements[0].Template.Name)
	require.Equal("air", schema.Statements[0].Template.Workspace.Package)
	require.Equal("Restaurant", schema.Statements[0].Template.Workspace.Name)
	require.Equal("wsTemplate_demo", schema.Statements[0].Template.Source)

	require.NotNil(schema.Statements[1].Function)
}

func Test_WorkspaceStatements(t *testing.T) {
	require := require.New(t)
	var schema = &schemaAST{}

	parser, err := participle.Build[schemaAST]()
	require.Nil(err)

	schema, err = parser.ParseString("", `
	SCHEMA test; 
	WORKSPACE MyWorkspace (
		FUNCTION MyFunc(param int) RETURNS void ENGINE WASM; 
		WORKSPACE ChildWorkspace (
			FUNCTION MyFunc2() RETURNS void ENGINE WASM; 
		)
	)
	`)
	require.Nil(err)
	require.Equal("test", schema.Package)
	require.Equal(1, len(schema.Statements))

	require.NotNil(schema.Statements[0].Workspace)
	ws := schema.Statements[0].Workspace
	require.Equal("MyWorkspace", ws.Name)
	require.Equal(2, len(ws.Statements))

	require.NotNil(ws.Statements[0].Function)
	require.Equal("MyFunc", ws.Statements[0].Function.Name)

	require.NotNil(ws.Statements[1].Workspace)
	require.Equal("ChildWorkspace", ws.Statements[1].Workspace.Name)
	require.Equal(1, len(ws.Statements[1].Workspace.Statements))

	require.NotNil(ws.Statements[1].Workspace.Statements[0].Function)
	require.Equal("MyFunc2", ws.Statements[1].Workspace.Statements[0].Function.Name)
}

func Test_Functions(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	parser, err := participle.Build[schemaAST]()
	require.Nil(err)

	ast, err = parser.ParseString("", `
	SCHEMA test;
	FUNCTION MyTableValidator() RETURNS void ENGINE BUILTIN;
	FUNCTION MyTableValidator(TableRow) RETURNS string ENGINE WASM;
	FUNCTION MyTableValidator(param1 aaa.TableRow, bbb.string) RETURNS ccc.TableRow ENGINE WASM;
	`)
	require.Nil(err)
	require.Equal("test", ast.Package)
	require.Equal(3, len(ast.Statements))

	s := ast.Statements[0]
	require.NotNil(s.Function)
	require.Equal("MyTableValidator", s.Function.Name)
	require.Nil(s.Function.Params)
	require.NotNil(s.Function.Returns)
	require.Equal("", s.Function.Returns.Package)
	require.Equal("void", s.Function.Returns.Name)
	require.True(s.Function.Engine.Builtin)
	require.False(s.Function.Engine.WASM)

	s = ast.Statements[1]
	require.Equal("MyTableValidator", s.Function.Name)
	require.NotNil(s.Function.Params)
	require.Equal(1, len(s.Function.Params))
	require.Nil(s.Function.Params[0].NamedParam)
	require.NotNil(s.Function.Params[0].UnnamedParamType)
	require.Equal("", s.Function.Params[0].UnnamedParamType.Package)
	require.Equal("TableRow", s.Function.Params[0].UnnamedParamType.Name)

	s = ast.Statements[2]
	require.Equal("MyTableValidator", s.Function.Name)
	require.NotNil(s.Function.Params)
	require.Equal(2, len(s.Function.Params))
	require.Nil(s.Function.Params[0].UnnamedParamType)
	require.NotNil(s.Function.Params[0].NamedParam)
	require.Equal("param1", s.Function.Params[0].NamedParam.Name)
	require.Equal("aaa", s.Function.Params[0].NamedParam.Type.Package)
	require.Equal("TableRow", s.Function.Params[0].NamedParam.Type.Name)

	require.Nil(s.Function.Params[1].NamedParam)
	require.NotNil(s.Function.Params[1].UnnamedParamType)
	require.Equal("bbb", s.Function.Params[1].UnnamedParamType.Package)
	require.Equal("string", s.Function.Params[1].UnnamedParamType.Name)

}

func Test_Projectors(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	// TODO: UPDATE OR INSERT OR DELETE? (UPDATE, INSERT, DEACTIVATE)?
	ast, err := ParseString2(`
	SCHEMA test;
	WORKSPACE MyWs (
		PROJECTOR ON COMMAND air.CreateUPProfile AS SomeFunc;
		PROJECTOR ON COMMAND ARGUMENT untill.QNameOrders AS xyz.SomeFunc2;
		PROJECTOR ON INSERT untill.bill AS SomeFunc;
		PROJECTOR ON INSERT OR UPDATE untill.bill AS SomeFunc;
		PROJECTOR ON UPDATE untill.bill AS SomeFunc;
		PROJECTOR ON UPDATE OR INSERT untill.bill AS SomeFunc;
		PROJECTOR ApplyUPProfile ON COMMAND IN (air.CreateUPProfile, air.UpdateUPProfile) AS air.FillUPProfile;
	)
	`)

	require.Nil(err)

	ws := ast.Statements[0].Workspace
	require.Equal(7, len(ws.Statements))

	p := ws.Statements[0].Projector
	require.Equal("", p.Name)
	require.Equal("COMMAND", p.On)
	require.Equal(1, len(p.Targets))
	require.Equal("air", p.Targets[0].Package)
	require.Equal("CreateUPProfile", p.Targets[0].Name)
	require.Equal("SomeFunc", p.Func.Name)
	require.Equal("", p.Func.Package)

	p = ws.Statements[1].Projector
	require.Equal("", p.Name)
	require.Equal("COMMANDARGUMENT", p.On)
	require.Equal(1, len(p.Targets))
	require.Equal("untill", p.Targets[0].Package)
	require.Equal("QNameOrders", p.Targets[0].Name)
	require.Equal("SomeFunc2", p.Func.Name)
	require.Equal("xyz", p.Func.Package)

	p = ws.Statements[2].Projector
	require.Equal("", p.Name)
	require.Equal("INSERT", p.On)
	require.Equal(1, len(p.Targets))
	require.Equal("untill", p.Targets[0].Package)
	require.Equal("bill", p.Targets[0].Name)
	require.Equal("SomeFunc", p.Func.Name)
	require.Equal("", p.Func.Package)

	p = ws.Statements[3].Projector
	require.Equal("", p.Name)
	require.Equal("INSERTORUPDATE", p.On)
	require.Equal(1, len(p.Targets))
	require.Equal("untill", p.Targets[0].Package)
	require.Equal("bill", p.Targets[0].Name)
	require.Equal("SomeFunc", p.Func.Name)
	require.Equal("", p.Func.Package)

	p = ws.Statements[4].Projector
	require.Equal("", p.Name)
	require.Equal("UPDATE", p.On)
	require.Equal(1, len(p.Targets))
	require.Equal("untill", p.Targets[0].Package)
	require.Equal("bill", p.Targets[0].Name)
	require.Equal("SomeFunc", p.Func.Name)
	require.Equal("", p.Func.Package)

	p = ws.Statements[5].Projector
	require.Equal("", p.Name)
	require.Equal("UPDATEORINSERT", p.On)
	require.Equal(1, len(p.Targets))
	require.Equal("untill", p.Targets[0].Package)
	require.Equal("bill", p.Targets[0].Name)
	require.Equal("SomeFunc", p.Func.Name)
	require.Equal("", p.Func.Package)

	p = ws.Statements[6].Projector
	require.Equal("ApplyUPProfile", p.Name)
	require.Equal("COMMAND", p.On)
	require.Equal(2, len(p.Targets))
	require.Equal("air", p.Targets[0].Package)
	require.Equal("CreateUPProfile", p.Targets[0].Name)
	require.Equal("air", p.Targets[1].Package)
	require.Equal("UpdateUPProfile", p.Targets[1].Name)
	require.Equal("FillUPProfile", p.Func.Name)
	require.Equal("air", p.Func.Package)
}

func Test_Grants(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	WORKSPACE MyWs (
		GRANT ALL ON ALL TABLES WITH TAG untill.Backoffice TO LocationManager;
    	GRANT INSERT,UPDATE ON ALL TABLES WITH TAG sys.ODoc TO LocationUser;
    	GRANT SELECT ON TABLE untill.orders TO LocationUser;
    	GRANT EXECUTE ON COMMAND Orders TO LocationUser;
    	GRANT EXECUTE ON QUERY TransactionHistory TO LocationUser;
    	GRANT EXECUTE ON ALL QUERIES WITH TAG PosTag TO LocationUser;
	)
	`)

	require.Nil(err)

	ws := ast.Statements[0].Workspace
	require.Equal(6, len(ws.Statements))

	p := ws.Statements[0].Grant
	require.Equal(1, len(p.Grants))
	require.Equal("ALL", p.Grants[0])
	require.Equal("ALLTABLESWITHTAG", p.On)
	require.Equal("untill", p.Target.Package)
	require.Equal("Backoffice", p.Target.Name)
	require.Equal("LocationManager", p.To)

	p = ws.Statements[1].Grant
	require.Equal(2, len(p.Grants))
	require.Equal("INSERT", p.Grants[0])
	require.Equal("UPDATE", p.Grants[1])
	require.Equal("ALLTABLESWITHTAG", p.On)
	require.Equal("sys", p.Target.Package)
	require.Equal("ODoc", p.Target.Name)
	require.Equal("LocationUser", p.To)

	p = ws.Statements[2].Grant
	require.Equal(1, len(p.Grants))
	require.Equal("SELECT", p.Grants[0])
	require.Equal("TABLE", p.On)
	require.Equal("untill", p.Target.Package)
	require.Equal("orders", p.Target.Name)
	require.Equal("LocationUser", p.To)

	p = ws.Statements[3].Grant
	require.Equal(1, len(p.Grants))
	require.Equal("EXECUTE", p.Grants[0])
	require.Equal("COMMAND", p.On)
	require.Equal("", p.Target.Package)
	require.Equal("Orders", p.Target.Name)
	require.Equal("LocationUser", p.To)

	p = ws.Statements[4].Grant
	require.Equal(1, len(p.Grants))
	require.Equal("EXECUTE", p.Grants[0])
	require.Equal("QUERY", p.On)
	require.Equal("", p.Target.Package)
	require.Equal("TransactionHistory", p.Target.Name)
	require.Equal("LocationUser", p.To)

	p = ws.Statements[5].Grant
	require.Equal(1, len(p.Grants))
	require.Equal("EXECUTE", p.Grants[0])
	require.Equal("ALLQUERIESWITHTAG", p.On)
	require.Equal("", p.Target.Package)
	require.Equal("PosTag", p.Target.Name)
	require.Equal("LocationUser", p.To)
}

func Test_Roles(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	ROLE UntillPaymentsUser;
	WORKSPACE MyWs (
		ROLE LocationManager;
	)
	`)

	require.Nil(err)

	role1 := ast.Statements[0].Role
	require.Equal("UntillPaymentsUser", role1.Name)

	ws := ast.Statements[1].Workspace
	require.Equal(1, len(ws.Statements))

	role2 := ws.Statements[0].Role
	require.Equal("LocationManager", role2.Name)
}

func Test_UseTable(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	WORKSPACE MyWs (
		USE TABLE somepackage.sometable;
		USE TABLE mytable;
		USE TABLE untill.*; 
	)
	`)

	require.Nil(err)

	ws := ast.Statements[0].Workspace

	u := ws.Statements[0].UseTable
	require.Equal("somepackage", u.Table.Package)
	require.Equal("sometable", u.Table.Name)

	u = ws.Statements[1].UseTable
	require.Equal("", u.Table.Package)
	require.Equal("mytable", u.Table.Name)
	require.False(u.Table.AllTables)

	u = ws.Statements[2].UseTable
	require.Equal("untill", u.Table.Package)
	require.Equal("", u.Table.Name)
	require.True(u.Table.AllTables)

}

func Test_Tags(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	TAG Backoffice;
	WORKSPACE MyWs (
		TAG Pos;
	)
	`)

	require.Nil(err)

	v := ast.Statements[0].Tag
	require.Equal("Backoffice", v.Name)

	ws := ast.Statements[1].Workspace
	require.Equal(1, len(ws.Statements))

	v = ws.Statements[0].Tag
	require.Equal("Pos", v.Name)
}

func Test_Comments(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	COMMENT BackofficeComment "This is a backoffice tool";
	WORKSPACE MyWs (
		COMMENT PosComment "This is a POS tool";
	)
	`)

	require.Nil(err)

	v := ast.Statements[0].Comment
	require.Equal("BackofficeComment", v.Name)
	require.Equal("\"This is a backoffice tool\"", v.Value)

	ws := ast.Statements[1].Workspace
	require.Equal(1, len(ws.Statements))

	v = ws.Statements[0].Comment
	require.Equal("PosComment", v.Name)
	require.Equal("\"This is a POS tool\"", v.Value)
}

func Test_Sequence(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	SEQUENCE bill_numbers int START WITH 1;
	SEQUENCE bill_numbers2 int MINVALUE 1;
	SEQUENCE SomeDecrementSeqneuce int MAXVALUE 1000000 INCREMENT BY -1;
	`)

	require.Nil(err)

	v := ast.Statements[0].Sequence
	require.Equal("bill_numbers", v.Name)
	require.Equal("int", v.Type)
	require.Equal(1, *v.StartWith)

	v = ast.Statements[1].Sequence
	require.Equal("bill_numbers2", v.Name)
	require.Equal("int", v.Type)
	require.Equal(1, *v.MinValue)
	require.False(v.Decrement)

	v = ast.Statements[2].Sequence
	require.Equal("SomeDecrementSeqneuce", v.Name)
	require.Equal("int", v.Type)
	require.Equal(1000000, *v.MaxValue)
	require.Equal(1, *v.IncrementBy)
	require.True(v.Decrement)
}

func Test_Rate(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	WORKSPACE MyWs (
		RATE BackofficeFuncRate1 1000 PER HOUR;
		RATE BackofficeFuncRate2 100 PER MINUTE PER IP;
	)
	`)

	require.Nil(err)

	ws := ast.Statements[0].Workspace
	require.Equal(2, len(ws.Statements))

	v := ws.Statements[0].Rate
	require.Equal("BackofficeFuncRate1", v.Name)
	require.Equal(1000, v.Amount)
	require.Equal("HOUR", v.Per)
	require.Equal(false, v.PerIP)

	v = ws.Statements[1].Rate
	require.Equal("BackofficeFuncRate2", v.Name)
	require.Equal(100, v.Amount)
	require.Equal("MINUTE", v.Per)
	require.Equal(true, v.PerIP)
}

func Test_Commands(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	WORKSPACE MyWs (
		COMMAND Orders AS PbillFunc;
		COMMAND Orders() AS PbillFunc WITH Comment=air.PosComment, Tags=[Tag1, air.Tag2];
		COMMAND Orders2(untill.orders) AS PbillFunc;
		COMMAND Orders3(order untill.orders, untill.pbill) AS PbillFunc;
	)
	`)

	require.Nil(err)

	ws := ast.Statements[0].Workspace
	require.Equal(4, len(ws.Statements))

	cmd := ws.Statements[0].Command
	require.Equal("Orders", cmd.Name)
	require.Equal(0, len(cmd.Params))
	require.Equal("PbillFunc", cmd.Func)

	cmd = ws.Statements[1].Command
	require.Equal("Orders", cmd.Name)
	require.Equal(0, len(cmd.Params))
	require.Equal("PbillFunc", cmd.Func)
	require.Equal(2, len(cmd.With))
	require.Equal("air", cmd.With[0].Comment.Package)
	require.Equal("PosComment", cmd.With[0].Comment.Name)
	require.Equal(0, len(cmd.With[0].Tags))
	require.Nil(cmd.With[1].Comment)
	require.Equal(2, len(cmd.With[1].Tags))
	require.Equal("", cmd.With[1].Tags[0].Package)
	require.Equal("Tag1", cmd.With[1].Tags[0].Name)
	require.Equal("air", cmd.With[1].Tags[1].Package)
	require.Equal("Tag2", cmd.With[1].Tags[1].Name)

	cmd = ws.Statements[2].Command
	require.Equal("Orders2", cmd.Name)
	require.Equal(1, len(cmd.Params))
	require.Nil(cmd.Params[0].NamedParam)
	require.NotNil(cmd.Params[0].UnnamedParamType)
	require.Equal("untill", cmd.Params[0].UnnamedParamType.Package)
	require.Equal("orders", cmd.Params[0].UnnamedParamType.Name)
	require.Equal("PbillFunc", cmd.Func)

	cmd = ws.Statements[3].Command
	require.Equal("Orders3", cmd.Name)
	require.Equal(2, len(cmd.Params))
	require.NotNil(cmd.Params[0].NamedParam)
	require.Nil(cmd.Params[0].UnnamedParamType)
	require.Equal("order", cmd.Params[0].NamedParam.Name)
	require.Equal("orders", cmd.Params[0].NamedParam.Type.Name)
	require.Equal("untill", cmd.Params[0].NamedParam.Type.Package)

	require.Nil(cmd.Params[1].NamedParam)
	require.NotNil(cmd.Params[1].UnnamedParamType)
	require.Equal("untill", cmd.Params[1].UnnamedParamType.Package)
	require.Equal("pbill", cmd.Params[1].UnnamedParamType.Name)
	require.Equal("PbillFunc", cmd.Func)
}

func Test_Queries(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	WORKSPACE MyWs (
		QUERY Query1 RETURNS QueryResellerInfoResult AS PbillFunc;
		QUERY Query1() RETURNS air.QueryResellerInfoResult AS PbillFunc WITH Comment=air.PosComment, Tags=[Tag1, air.Tag2];
		QUERY Query2(untill.orders) RETURNS QueryResellerInfoResult AS PbillFunc;
		QUERY Query3(order untill.orders, untill.pbill) RETURNS QueryResellerInfoResult AS PbillFunc;
	)
	`)

	require.Nil(err)

	ws := ast.Statements[0].Workspace
	require.Equal(4, len(ws.Statements))

	q := ws.Statements[0].Query
	require.Equal("Query1", q.Name)
	require.Equal("QueryResellerInfoResult", q.Returns.Name)
	require.Equal("", q.Returns.Package)
	require.Equal(0, len(q.Params))
	require.Equal("PbillFunc", q.Func)

	q = ws.Statements[1].Query
	require.Equal("Query1", q.Name)
	require.Equal("QueryResellerInfoResult", q.Returns.Name)
	require.Equal("air", q.Returns.Package)
	require.Equal(0, len(q.Params))
	require.Equal("PbillFunc", q.Func)
	require.Equal(2, len(q.With))
	require.Equal("air", q.With[0].Comment.Package)
	require.Equal("PosComment", q.With[0].Comment.Name)
	require.Equal(0, len(q.With[0].Tags))
	require.Nil(q.With[1].Comment)
	require.Equal(2, len(q.With[1].Tags))
	require.Equal("", q.With[1].Tags[0].Package)
	require.Equal("Tag1", q.With[1].Tags[0].Name)
	require.Equal("air", q.With[1].Tags[1].Package)
	require.Equal("Tag2", q.With[1].Tags[1].Name)

	q = ws.Statements[2].Query
	require.Equal("Query2", q.Name)
	require.Equal("QueryResellerInfoResult", q.Returns.Name)
	require.Equal(1, len(q.Params))
	require.Nil(q.Params[0].NamedParam)
	require.NotNil(q.Params[0].UnnamedParamType)
	require.Equal("untill", q.Params[0].UnnamedParamType.Package)
	require.Equal("orders", q.Params[0].UnnamedParamType.Name)
	require.Equal("PbillFunc", q.Func)

	q = ws.Statements[3].Query
	require.Equal("Query3", q.Name)
	require.Equal("QueryResellerInfoResult", q.Returns.Name)
	require.Equal(2, len(q.Params))
	require.NotNil(q.Params[0].NamedParam)
	require.Nil(q.Params[0].UnnamedParamType)
	require.Equal("order", q.Params[0].NamedParam.Name)
	require.Equal("orders", q.Params[0].NamedParam.Type.Name)
	require.Equal("untill", q.Params[0].NamedParam.Type.Package)

	require.Nil(q.Params[1].NamedParam)
	require.NotNil(q.Params[1].UnnamedParamType)
	require.Equal("untill", q.Params[1].UnnamedParamType.Package)
	require.Equal("pbill", q.Params[1].UnnamedParamType.Name)

	require.Equal("PbillFunc", q.Func)

}

func Test_Tables(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	TABLE air_table_plan OF CDOC (
        fstate int,
        name text NOT NULL,
		vf text NOT NULL VERIFIABLE,
		i1 int DEFAULT 1,
		s1 text DEFAULT "a",
		ii int DEFAULT NEXTVAL(sequence),
		id_bill int64 REFERENCES air.bill,
		ckf text CHECK "^[0-9]{8}$",
		UNIQUE fstate, name
    ) WITH Comment=air.PosComment, Tags=[Tag1, air.Tag2];
	WORKSPACE ws(
		TABLE ws_table OF CDOC, air.SomeType (
			psname text,
			TABLE child (
				number int				
			)
		);	
	)
	`)

	require.Nil(err)

	v := ast.Statements[0].Table
	require.Equal("air_table_plan", v.Name)
	require.Equal(1, len(v.Of))
	require.Equal("CDOC", v.Of[0].Name)
	require.Equal("", v.Of[0].Package)

	f := v.Items[0].Field
	require.Equal("fstate", f.Name)
	require.Equal("int", f.Type.Name)
	require.Equal("", f.Type.Package)
	require.False(f.NotNull)
	require.False(f.Verifiable)

	f = v.Items[1].Field
	require.Equal("name", f.Name)
	require.Equal("text", f.Type.Name)
	require.Equal("", f.Type.Package)
	require.True(f.NotNull)
	require.False(f.Verifiable)

	f = v.Items[2].Field
	require.Equal("vf", f.Name)
	require.Equal("text", f.Type.Name)
	require.Equal("", f.Type.Package)
	require.Nil(f.DefaultIntValue)
	require.Nil(f.DefaultStringValue)
	require.Nil(f.DefaultNextVal)
	require.Nil(f.References)
	require.Nil(f.CheckRegexp)
	require.True(f.NotNull)
	require.True(f.Verifiable)

	f = v.Items[3].Field
	require.Equal("i1", f.Name)
	require.Equal("int", f.Type.Name)
	require.NotNil(f.DefaultIntValue)
	require.Equal(1, *f.DefaultIntValue)

	f = v.Items[4].Field
	require.Equal("s1", f.Name)
	require.Equal("text", f.Type.Name)
	require.NotNil(f.DefaultStringValue)
	require.Equal("\"a\"", *f.DefaultStringValue)

	f = v.Items[5].Field
	require.Equal("ii", f.Name)
	require.Equal("int", f.Type.Name)
	require.NotNil(f.DefaultNextVal)
	require.Equal("sequence", *f.DefaultNextVal)

	f = v.Items[6].Field
	require.Equal("id_bill", f.Name)
	require.Equal("int64", f.Type.Name)
	require.NotNil(f.References)
	require.Equal("bill", f.References.Name)
	require.Equal("air", f.References.Package)

	f = v.Items[7].Field
	require.Equal("ckf", f.Name)
	require.Equal("text", f.Type.Name)
	require.NotNil(f.CheckRegexp)
	require.Equal("\"^[0-9]{8}$\"", *f.CheckRegexp)

	u := v.Items[8].Unique
	require.Equal(2, len(u.Fields))
	require.Equal("fstate", u.Fields[0])
	require.Equal("name", u.Fields[1])

	require.Equal(2, len(v.With))
	require.Equal("air", v.With[0].Comment.Package)
	require.Equal("PosComment", v.With[0].Comment.Name)
	require.Equal(0, len(v.With[0].Tags))
	require.Nil(v.With[1].Comment)
	require.Equal(2, len(v.With[1].Tags))
	require.Equal("", v.With[1].Tags[0].Package)
	require.Equal("Tag1", v.With[1].Tags[0].Name)
	require.Equal("air", v.With[1].Tags[1].Package)
	require.Equal("Tag2", v.With[1].Tags[1].Name)

	ws := ast.Statements[1].Workspace
	v = ws.Statements[0].Table
	require.Equal("ws_table", v.Name)
	require.Equal(2, len(v.Of))
	require.Equal("CDOC", v.Of[0].Name)
	require.Equal("", v.Of[0].Package)
	require.Equal("SomeType", v.Of[1].Name)
	require.Equal("air", v.Of[1].Package)

	require.Equal(2, len(v.Items))
	require.NotNil(v.Items[0].Field)
	require.Equal("psname", v.Items[0].Field.Name)
	require.Equal("text", v.Items[0].Field.Type.Name)
	require.Equal("", v.Items[0].Field.Type.Package)

	require.NotNil(v.Items[1].Table)
	child := v.Items[1].Table
	require.Equal("child", child.Name)
	require.Equal(0, len(child.Of))
	require.Equal(1, len(child.Items))
	require.Equal("number", child.Items[0].Field.Name)
	require.Equal("int", child.Items[0].Field.Type.Name)
	require.Equal("", child.Items[0].Field.Type.Package)
}

func Test_ViewsAsResultOfProjectors(t *testing.T) {
	require := require.New(t)
	var ast = &schemaAST{}

	ast, err := ParseString(`
	SCHEMA test;
	WORKSPACE MyWs (
		VIEW XZReports(
			Year int32,
			Month int32, 
			Day int32, 
			Kind int32, 
			Number int32, 
			XZReportWDocID id
		) AS RESULT OF air.UpdateXZReportsView
		WITH 
			PrimaryKey="(Year), Month, Day, Kind, Number",
			Comment=PosComment
	)
	`)

	require.Nil(err)

	ws := ast.Statements[0].Workspace

	v := ws.Statements[0].View
	require.Equal("XZReports", v.Name)
	require.Equal(6, len(v.Fields))
	require.Equal("Year", v.Fields[0].Name)
	require.Equal("int32", v.Fields[0].Type)
	require.Equal("air", v.ResultOf.Package)
	require.Equal("UpdateXZReportsView", v.ResultOf.Name)
	require.Equal(2, len(v.With))
	require.NotNil(v.With[0].PrimaryKey)
	require.Equal("\"(Year), Month, Day, Kind, Number\"", *v.With[0].PrimaryKey)

	require.NotNil(v.With[1].Comment)
	require.Equal("", v.With[1].Comment.Package)
	require.Equal("PosComment", v.With[1].Comment.Name)

}
