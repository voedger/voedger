// Copyright (c) 2021-present Voedger Authors.
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// @author Michael Saigachenko

package pipeline

import (
	"fmt"
)

func pipelinePanic(msg string, operatorName string, context IWorkpieceContext) {
	panic(fmt.Sprintf("critical error in operator '%s': %s. Pipeline '%s' [%s]", operatorName, msg, context.GetPipelineName(), context.GetPipelineStruct()))
}
