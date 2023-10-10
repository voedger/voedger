/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Funcs struct {
	Commands map[appdef.QName]*CommandFunc `json:",omitempty"`
	Queries  map[appdef.QName]*QueryFunc   `json:",omitempty"`
}

type Func struct {
	Comment   string `json:",omitempty"`
	Name      appdef.QName
	Arg       *appdef.QName `json:",omitempty"`
	Result    *appdef.QName `json:",omitempty"`
	Extension Extension
}

type Extension struct {
	Name   string
	Engine appdef.ExtensionEngineKind
}

type CommandFunc struct {
	Func
	UnloggedArg *appdef.QName `json:",omitempty"`
}

type QueryFunc struct {
	Func
}
