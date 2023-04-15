/*
*
* Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*
 */

package pipeline

import (
	"fmt"
)

func pipelinePanic(msg string, operatorName string, context IWorkpieceContext) {
	panic(fmt.Sprintf("critical error in operator '%s': %s. Pipeline '%s' [%s]", operatorName, msg, context.GetPipelineName(), context.GetPipelineStruct()))
}
