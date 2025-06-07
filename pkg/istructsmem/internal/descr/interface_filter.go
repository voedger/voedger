/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Filter struct {
	QNames    []appdef.QName    `json:",omitempty"`
	Types     []appdef.TypeKind `json:",omitempty"`
	Workspace *appdef.QName     `json:",omitempty"`
	Tags      []appdef.QName    `json:",omitempty"`
	And       []*Filter         `json:",omitempty"`
	Or        []*Filter         `json:",omitempty"`
	Not       *Filter           `json:",omitempty"`
}
