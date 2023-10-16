/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Functions struct {
	Commands map[appdef.QName]*CommandFunction `json:",omitempty"`
	Queries  map[appdef.QName]*QueryFunction   `json:",omitempty"`
}

type Function struct {
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

type CommandFunction struct {
	Function
	UnloggedArg *appdef.QName `json:",omitempty"`
}

type QueryFunction struct {
	Function
}
