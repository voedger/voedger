/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "iter"

// # Supports:
//   - ITag
type tag struct {
	typ
}

// Creates and returns new tag.
func newTag(app *appDef, ws *workspace, name QName) *tag {
	t := &tag{
		typ: makeType(app, ws, name, TypeKind_Tag),
	}
	ws.appendType(t)
	return t
}

func (tag) isTag() {}

// # Supports:
//	 - IWithTags
type tags struct {
	find FindType
	list *types[ITag]
}

func makeTags(find FindType) tags {
	return tags{find, newTypes[ITag]()}
}

func (t *tags) HasTag(name QName) bool {
	return t.list.find(name) != NullType
}

func (t *tags) Tags() iter.Seq[ITag] {
	return t.list.all
}

// # Supports:
//   - ITagBuilder
type tagBuilder struct {
	*tags
}

func makeTagBuilder(tags *tags) tagBuilder {
	return tagBuilder{tags}
}

func (t *tagBuilder) SetTag(tag ...QName) {
	t.tags.list.clear()
	for _, name := range tag {
		tag := Tag(t.tags.find, name)
		if tag == nil {
			panic(ErrNotFound("tag %s", name))
		}
		t.tags.list.add(tag)
	}
}

type nullTags struct{}

func (t nullTags) HasTag(QName) bool    { return false }
func (t nullTags) Tags() iter.Seq[ITag] { return func(func(ITag) bool) {} }
