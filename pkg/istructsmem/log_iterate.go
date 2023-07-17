/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"github.com/voedger/voedger/pkg/istructs"
)

// logReadPartFuncType is function type to read log part (≤ 4096 events). Must return ok and nil error to read next part.
type logReadPartFuncType func(partID uint64, ccolsFrom, ccolsTo []byte) (ok bool, err error)

// readLogParts in a loop reads events from the log by parts (≤ 4096 events) by call readPart() function
func readLogParts(startOffset istructs.Offset, toReadCount int, readPart logReadPartFuncType) error {
	if toReadCount <= 0 {
		return nil
	}

	var finishOffset istructs.Offset
	if toReadCount == istructs.ReadToTheEnd {
		finishOffset = istructs.Offset(istructs.ReadToTheEnd)
	} else {
		finishOffset = startOffset + istructs.Offset(toReadCount)
	}

	minPart, _ := crackLogOffset(startOffset)
	maxPart, _ := crackLogOffset(finishOffset - 1)

	for part := minPart; part <= maxPart; part++ {
		ccolsFrom := []byte(nil)
		if part == minPart {
			_, ccolsFrom = splitLogOffset(startOffset)
		}
		ccolsTo := []byte(nil)
		if (part == maxPart) && (toReadCount != istructs.ReadToTheEnd) && (finishOffset%partitionRecordCount != 0) {
			_, ccolsTo = splitLogOffset(finishOffset)
		}

		ok, err := readPart(part, ccolsFrom, ccolsTo)

		if err != nil {
			return err
		}
		if !ok {
			break
		}
	}

	return nil
}
