/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package apppartsctl_test

import "github.com/voedger/voedger/pkg/appparts"

type ProcMock struct {
	k appparts.ProcKind
}

func (p ProcMock) String() string { return p.k.TrimString() }

func MockProcessors(commands, queries, projectors int) [appparts.ProcKind_Count][]appparts.IProc {
	p := [appparts.ProcKind_Count][]appparts.IProc{
		appparts.ProcKind_Command:   make([]appparts.IProc, commands, commands),
		appparts.ProcKind_Query:     make([]appparts.IProc, queries, queries),
		appparts.ProcKind_Projector: make([]appparts.IProc, projectors, projectors),
	}
	for i := 0; i < commands; i++ {
		p[appparts.ProcKind_Command][i] = ProcMock{appparts.ProcKind_Command}
	}
	for i := 0; i < queries; i++ {
		p[appparts.ProcKind_Query][i] = ProcMock{appparts.ProcKind_Query}
	}
	for i := 0; i < projectors; i++ {
		p[appparts.ProcKind_Projector][i] = ProcMock{appparts.ProcKind_Projector}
	}
	return p
}
