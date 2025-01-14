/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func (k ProcessorKind) MarshalText() ([]byte, error) {
	var s string
	if k < ProcessorKind_Count {
		s = k.String()
	} else {
		s = utils.UintToString(k)
	}
	return []byte(s), nil
}

// Renders an ProcessorKind in human-readable form, without `ProcessorKind_` prefix,
// suitable for debugging or error messages
func (k ProcessorKind) TrimString() string {
	const pref = "ProcessorKind_"
	return strings.TrimPrefix(k.String(), pref)
}

// Returns is the processor kind compatible with the extension and an error if not
func (k ProcessorKind) CompatibleWithExtension(ext appdef.IExtension) (bool, error) {
	t := ext.Kind()
	switch k {
	case ProcessorKind_Command:
		if t == appdef.TypeKind_Command {
			return true, nil
		}
		if t == appdef.TypeKind_Projector {
			if prj, ok := ext.(appdef.IProjector); ok && prj.Sync() {
				return true, nil
			}
		}
	case ProcessorKind_Query:
		if t == appdef.TypeKind_Query {
			return true, nil
		}
	case ProcessorKind_Actualizer:
		if t == appdef.TypeKind_Projector {
			if prj, ok := ext.(appdef.IProjector); ok && !prj.Sync() {
				return true, nil
			}
		}
	case ProcessorKind_Scheduler:
		if t == appdef.TypeKind_Job {
			return true, nil
		}
	}
	return false, errExtensionIncompatibleWithProcessor(ext, k)
}
