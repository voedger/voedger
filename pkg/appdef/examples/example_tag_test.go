/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/builder"
)

func ExampleTags() {

	var app appdef.IAppDef

	wsName := appdef.NewQName("test", "workspace")
	tagNames := appdef.QNamesFrom(appdef.NewQName("test", "tag1"), appdef.NewQName("test", "tag2"), appdef.NewQName("test", "unusedTag"))
	objName := appdef.NewQName("test", "object")

	// how to build AppDef with tags
	{
		adb := builder.New()
		adb.AddPackage("test", "test.com/test")

		wsb := adb.AddWorkspace(wsName)

		wsb.AddTag(tagNames[0], "first tag comment")
		wsb.AddTag(tagNames[1], "second tag comment")

		wsb.AddTag(tagNames[2], "unused tag comment")

		obj := wsb.AddObject(objName)
		obj.SetTag(tagNames[0], tagNames[1])

		app = adb.MustBuild()
	}

	// how to find tag in builded AppDef
	{
		fmt.Println("Find tag in application:")
		tag := appdef.Tag(app.Type, tagNames[0])
		fmt.Println("-", tag, tag.Comment())
		fmt.Println("-", appdef.Tag(app.Type, appdef.NewQName("test", "unknown")))
	}

	// How to enum all tags in AppDef
	{
		fmt.Println("All application tags:")
		for tag := range appdef.Tags(app.Types()) {
			fmt.Println("-", tag, tag.Comment())
		}
	}

	// How to check if type has tag
	{
		obj := appdef.Object(app.Type, objName)
		fmt.Println(obj, "has tags:")
		for _, tag := range tagNames {
			fmt.Println("-", tag, obj.HasTag(tag))
		}
	}

	// How to enum all type tags
	{
		obj := appdef.Object(app.Type, objName)
		fmt.Println(obj, "tags:")
		for _, tag := range obj.Tags() {
			fmt.Println("-", tag, tag.Comment())
		}
	}

	// Output:
	// Find tag in application:
	// - Tag «test.tag1» first tag comment
	// - <nil>
	// All application tags:
	// - Tag «test.tag1» first tag comment
	// - Tag «test.tag2» second tag comment
	// - Tag «test.unusedTag» unused tag comment
	// Object «test.object» has tags:
	// - test.tag1 true
	// - test.tag2 true
	// - test.unusedTag false
	// Object «test.object» tags:
	// - Tag «test.tag1» first tag comment
	// - Tag «test.tag2» second tag comment
}
