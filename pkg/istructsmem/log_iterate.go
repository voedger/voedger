/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"github.com/voedger/voedger/pkg/istructs"
)

// Function type to read log part (≤ 4096 events). Must read all records offsets within closed range [ofsHi+ofsLo1 … ofsHi+ofsLo2]
//
// Must return ok and nil error to read next part
type logReadPartFuncType func(ofsHi uint64, ofsLo1, ofsLo2 uint16) (ok bool, err error)

// readLogParts in a loop reads events from the log by parts (≤ 4096 events) by call readPart() function
func readLogParts(startOffset istructs.Offset, toReadCount int, readPart logReadPartFuncType) error {
	if toReadCount <= 0 {
		return nil
	}

	var finishOffset istructs.Offset
	if toReadCount == istructs.ReadToTheEnd {
		finishOffset = istructs.Offset(istructs.ReadToTheEnd)
	} else {
		finishOffset = startOffset + istructs.Offset(toReadCount) - 1
	}

	minPart, _ := crackLogOffset(startOffset)
	maxPart, _ := crackLogOffset(finishOffset)

	for part := minPart; part <= maxPart; part++ {
		ccolsFrom := uint16(0)
		if part == minPart {
			_, ccolsFrom = crackLogOffset(startOffset)
		}
		ccolsTo := lowMask
		if (part == maxPart) && (toReadCount != istructs.ReadToTheEnd) && (finishOffset%partitionRecordCount != 0) {
			_, ccolsTo = crackLogOffset(finishOffset)
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
