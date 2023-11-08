/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts_test

import "github.com/voedger/voedger/pkg/appparts"

type EngineMock struct {
	k appparts.ProcKind
}

func (p EngineMock) String() string { return p.k.TrimString() }

func MockEngines(commands, queries, projectors int) [appparts.ProcKind_Count][]appparts.IEngine {
	ee := [appparts.ProcKind_Count][]appparts.IEngine{
		appparts.ProcKind_Command:   make([]appparts.IEngine, commands),
		appparts.ProcKind_Query:     make([]appparts.IEngine, queries),
		appparts.ProcKind_Projector: make([]appparts.IEngine, projectors),
	}
	for i := 0; i < commands; i++ {
		ee[appparts.ProcKind_Command][i] = EngineMock{appparts.ProcKind_Command}
	}
	for i := 0; i < queries; i++ {
		ee[appparts.ProcKind_Query][i] = EngineMock{appparts.ProcKind_Query}
	}
	for i := 0; i < projectors; i++ {
		ee[appparts.ProcKind_Projector][i] = EngineMock{appparts.ProcKind_Projector}
	}
	return ee
}
