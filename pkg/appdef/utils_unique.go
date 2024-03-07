/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "fmt"

// Constructs and returns QName for the unique for the document.
func UniqueQName(docQName QName, uniqueName string) QName {
	return NewQName(docQName.Pkg(), fmt.Sprintf("%s$uniques$%s", docQName.Entity(), uniqueName))
}
