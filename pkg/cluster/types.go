/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package cluster

// ProcKind is a enumeration of processors.
//
// Ref to proc-king.go for values and methods
type ProcKind uint8

//go:generate stringer -type=ProcKind

const (
	ProcKind_Command ProcKind = iota
	ProcKind_Query
	ProcKind_Projector

	ProcKind_Count
)

type AppDeploymentDescriptor struct {
	NumParts int

	EnginePoolSize [ProcKind_Count]int
}
