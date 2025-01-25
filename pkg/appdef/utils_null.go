/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

type nullComment struct{}

func (c *nullComment) Comment() string        { return "" }
func (c *nullComment) CommentLines() []string { return nil }

const nullTypeString = "null type"

type nullTags struct{}

func (t nullTags) HasTag(QName) bool { return false }
func (t nullTags) Tags() []ITag      { return nil }

type nullType struct {
	nullComment
	nullTags
}

func (t nullType) App() IAppDef          { return nil }
func (t nullType) IsSystem() bool        { return false }
func (t nullType) Kind() TypeKind        { return TypeKind_null }
func (t nullType) QName() QName          { return NullQName }
func (t nullType) String() string        { return nullTypeString }
func (t nullType) Workspace() IWorkspace { return nil }

type nullFields struct{}

func (f *nullFields) Field(FieldName) IField       { return nil }
func (f *nullFields) FieldCount() int              { return 0 }
func (f *nullFields) Fields() []IField             { return nil }
func (f *nullFields) RefField(FieldName) IRefField { return nil }
func (f *nullFields) RefFields() []IRefField       { return nil }
func (f *nullFields) UserFieldCount() int          { return 0 }
func (f *nullFields) UserFields() []IField         { return nil }
