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
	get := func(k appparts.ProcKind, cnt int) []appparts.IEngine {
		s := make([]appparts.IEngine, cnt)
		for i := 0; i < cnt; i++ {
			s[i] = EngineMock{k}
		}
		return s
	}
	return [appparts.ProcKind_Count][]appparts.IEngine{
		appparts.ProcKind_Command:   get(appparts.ProcKind_Command, commands),
		appparts.ProcKind_Query:     get(appparts.ProcKind_Query, queries),
		appparts.ProcKind_Projector: get(appparts.ProcKind_Projector, queries),
	}
}
