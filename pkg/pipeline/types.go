/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*
 */

package pipeline

type WorkpieceContext struct {
	pipelineName   string
	pipelineStruct string
}

func (c WorkpieceContext) GetPipelineName() string {
	return c.pipelineName
}

func (c WorkpieceContext) GetPipelineStruct() string {
	return c.pipelineStruct
}

func NewWorkpieceContext(pName, pStruct string) WorkpieceContext {
	return WorkpieceContext{
		pipelineName:   pName,
		pipelineStruct: pStruct,
	}
}

type BatchItem struct {
	Key   interface{}
	Value interface{}
}
