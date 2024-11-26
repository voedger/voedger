/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

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

// # Supports:
//	 - ITags
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

func (t *tags) Tags(visit func(ITag) bool) {
	t.list.all(visit)
}

// # Supports:
//   - ITagBuilder
type tagBuilder struct {
	*tags
}

func makeTagBuilder(tags *tags) tagBuilder {
	return tagBuilder{tags}
}

func (t *tagBuilder) SetTag(tag QName, tags ...QName) {
	add := func(name QName) {
		typ := t.tags.find(name)
		if typ.Kind() != TypeKind_Tag {
			panic(ErrNotFound("tag %s", name))
		}
		t.tags.list.add(typ.(ITag))
	}

	add(tag)
	for _, tag := range tags {
		add(tag)
	}
}

type nullTags struct{}

func (t nullTags) HasTag(QName) bool    { return false }
func (t nullTags) Tags(func(ITag) bool) {}
