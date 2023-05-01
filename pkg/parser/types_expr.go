/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package parser

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "TRUE"
	return nil
}

type Expression struct {
	Or []*OrCondition `parser:"@@ ( 'OR' @@ )*"`
}

type OrCondition struct {
	And []*Condition `parser:"@@ ( 'AND' @@ )*"`
}

type Condition struct {
	Operand *ConditionOperand `parser:"  @@"`
	Not     *Condition        `parser:"| 'NOT' @@"`
}

type ConditionOperand struct {
	Operand      *Operand      `parser:"@@"`
	ConditionRHS *ConditionRHS `parser:"@@?"`
}

type ConditionRHS struct {
	Compare *Compare `parser:"  @@"`
	Is      *Is      `parser:"| 'IS' @@"`
	Between *Between `parser:"| 'BETWEEN' @@"`
	In      *In      `parser:"| 'IN' '(' @@ ')'"`
}

type Compare struct {
	Operator string   `parser:"@( '<>' | '<=' | '>=' | '=' | '<' | '>' | '!=' )"`
	Operand  *Operand `parser:"@@"`
}

type Like struct {
	Not     bool     `parser:"[ @'NOT' ]"`
	Operand *Operand `parser:"@@"`
}

type Is struct {
	Not  bool `parser:"[ @'NOT' ]"`
	Null bool `parser:"@'NULL'"`
}

type Between struct {
	Start *Operand `parser:"@@"`
	End   *Operand `parser:"'AND' @@"`
}

type In struct {
	Expressions []*Expression `parser:"@@ ( ',' @@ )*"`
}

type Operand struct {
	LHS *Factor `parser:"@@"`
	Op  string  `parser:"[ @('+' | '-')"`
	RHS *Factor `parser:"  @@ ]"`
}

type Factor struct {
	LHS *Term  `parser:"@@"`
	Op  string `parser:"( @('*' | '/' | '%')"`
	RHS *Term  `parser:"  @@ )?"`
}

type Term struct {
	Value         *Value      `parser:"@@"`
	SymbolRef     *SymbolRef  `parser:"| @@"`
	SubExpression *Expression `parser:"| '(' @@ ')'"`
}

type SymbolRef struct {
	Name       DefQName      `parser:"@@"`
	Parameters []*Expression `parser:"( '(' @@ ( ',' @@ )* ')' )?"`
}

type Value struct {
	Int     *int64   `parser:" (@Int"`
	Float   *float64 `parser:" | @Float"`
	String  *string  `parser:" | @String"`
	Boolean *Boolean `parser:" | @('TRUE' | 'FALSE')"`
	Null    bool     `parser:" | @'NULL' )"`
}

type Array struct {
	Expressions []*Expression `parser:"'(' @@ ( ',' @@ )* ')'"`
}
