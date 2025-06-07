/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package types

import (
	"github.com/voedger/voedger/pkg/appdef"
)

// # Supports:
//   - appdef.ITag
type Tag struct {
	Typ
	feature string
}

// Creates and returns new tag.
func NewTag(ws appdef.IWorkspace, name appdef.QName, feature string) *Tag {
	t := &Tag{
		Typ:     MakeType(ws.App(), ws, name, appdef.TypeKind_Tag),
		feature: feature,
	}
	Propagate(t)
	return t
}

func (t Tag) Feature() string { return t.feature }

// # Supports:
//   - IWithTags
type WithTags struct {
	find appdef.FindType
	list *Types[appdef.ITag]
}

func MakeWithTags(find appdef.FindType) WithTags {
	return WithTags{find, NewTypes[appdef.ITag]()}
}

func (t WithTags) HasTag(name appdef.QName) bool {
	return t.list.Find(name) != appdef.NullType
}

func (t WithTags) Tags() []appdef.ITag {
	return t.list.AsArray()
}

func (t *WithTags) setTag(tag ...appdef.QName) {
	t.list.Clear()
	for _, name := range tag {
		tag := appdef.Tag(t.find, name)
		if tag == nil {
			panic(appdef.ErrNotFound("tag %s", name))
		}
		t.list.Add(tag)
	}
}

// # Supports:
//   - appdef.ITagBuilder
type TagBuilder struct {
	*WithTags
}

func MakeTagBuilder(tags *WithTags) TagBuilder {
	return TagBuilder{tags}
}

func (t *TagBuilder) SetTag(tag ...appdef.QName) { t.setTag(tag...) }
