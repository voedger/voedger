/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

import "github.com/voedger/voedger/pkg/appdef"

type Extensions struct {
	Commands   map[appdef.QName]*CommandFunction `json:",omitempty"`
	Queries    map[appdef.QName]*QueryFunction   `json:",omitempty"`
	Projectors map[appdef.QName]*Projector       `json:",omitempty"`
	Jobs       map[appdef.QName]*Job             `json:",omitempty"`
}

type Extension struct {
	Type
	Name    string
	Engine  string
	States  map[appdef.QName]appdef.QNames `json:",omitempty"`
	Intents map[appdef.QName]appdef.QNames `json:",omitempty"`
}

type Function struct {
	Extension
	Arg    *appdef.QName `json:",omitempty"`
	Result *appdef.QName `json:",omitempty"`
}

type CommandFunction struct {
	Function
	UnloggedArg *appdef.QName `json:",omitempty"`
}

type QueryFunction struct {
	Function
}

type Projector struct {
	Extension
	Events     map[appdef.QName]ProjectorEvent `json:",omitempty"`
	WantErrors bool                            `json:",omitempty"`
}

type ProjectorEvent struct {
	Comment string       `json:",omitempty"`
	On      appdef.QName `json:"-"`
	Kind    []string     `json:",omitempty"`
}

type Job struct {
	Extension
	CronSchedule string
}
