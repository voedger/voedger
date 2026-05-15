/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appparts

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

var ErrNotFound = errors.New("not found")

// ErrDeployment is returned when application deployment fails because of a
// validation or extension-engine initialization error.
var ErrDeployment = errors.New("deployment error")

func errExtensionInVSQLNotInCode(app appdef.AppQName, ext appdef.IExtension, fqn appdef.FullQName) error {
	return fmt.Errorf("%w: app %v: %s %v (%v): in vsql, not in code", ErrDeployment, app, ext.Kind().TrimString(), ext.QName(), fqn)
}

func errExtensionInCodeNotInVSQL(app appdef.AppQName, fqn appdef.FullQName) error {
	return fmt.Errorf("%w: app %v: %v: in code, not in vsql", ErrDeployment, app, fqn)
}

func errExtensionUnknownPackage(app appdef.AppQName, ext appdef.IExtension) error {
	return fmt.Errorf("%w: app %v: %s %v: package «%s» full path is unknown", ErrDeployment, app, ext.Kind().TrimString(), ext.QName(), ext.QName().Pkg())
}

func errExtensionEngineDeploy(app appdef.AppQName, kind appdef.ExtensionEngineKind, err error) error {
	return fmt.Errorf("%w: app %v: extension engine %s: %w", ErrDeployment, app, kind, err)
}

var (
	ErrNotAvailableEngines                            = errors.New("no available engines")
	errNotAvailableEngines [ProcessorKind_Count]error = [ProcessorKind_Count]error{
		fmt.Errorf("%w %s", ErrNotAvailableEngines, ProcessorKind_Command.TrimString()),
		fmt.Errorf("%w %s", ErrNotAvailableEngines, ProcessorKind_Query.TrimString()),
		fmt.Errorf("%w %s", ErrNotAvailableEngines, ProcessorKind_Actualizer.TrimString()),
		fmt.Errorf("%w %s", ErrNotAvailableEngines, ProcessorKind_Scheduler.TrimString()),
	}
)

func errAppCannotBeRedeployed(name appdef.AppQName) error {
	return fmt.Errorf("application %v can not be redeployed: %w", name, errors.ErrUnsupported)
}

func errAppNotFound(name appdef.AppQName) error {
	return fmt.Errorf("application %v not found: %w", name, ErrNotFound)
}

func errPartitionNotFound(name appdef.AppQName, partID istructs.PartitionID) error {
	return fmt.Errorf("application %v partition %v not found: %w", name, partID, ErrNotFound)
}

func errUndefinedExtension(n appdef.QName) error {
	return fmt.Errorf("undefined extension %v: %w", n, ErrNotFound)
}

func errExtensionIncompatibleWithProcessor(ext appdef.IExtension, proc ProcessorKind) error {
	return fmt.Errorf("extension %v is not compatible with processor %v", ext, proc.TrimString())
}

func errCantObtainFullQName(n appdef.QName) error {
	return fmt.Errorf("can't obtain full qualified name for «%v»: %w", n, ErrNotFound)
}
