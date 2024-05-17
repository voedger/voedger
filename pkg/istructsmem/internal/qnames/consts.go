/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import "github.com/voedger/voedger/pkg/istructsmem/internal/vers"

// IDs for wellknown QNames
const (
	NullQNameID QNameID = 0 + iota
	QNameIDForError
	QNameIDCommandCUD
	QNameIDForCorruptedData

	QNameIDSysLast QNameID = 0xFF
)

// maximum QName ID value
const MaxAvailableQNameID = 0xFFFF

// QNames system view versions
const (
	ver01 vers.VersionValue = vers.UnknownVersion + 1

	latestVersion = ver01
)
