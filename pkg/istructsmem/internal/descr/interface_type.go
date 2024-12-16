/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Type struct {
	Comment string          `json:",omitempty"`
	QName   appdef.QName    `json:"-"`
	Kind    appdef.TypeKind `json:"-"`
	Tags    appdef.QNames   `json:",omitempty"`
}
