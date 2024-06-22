/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package projectors

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
)

// actualizers is a set of actualizers for application partitions.
//
// # Implements:
//   - IActualizers
//   - appparts.IActualizers
type actualizers struct {
	cfg  BasicAsyncActualizerConfig
	apps map[appdef.AppQName]struct {
		parts map[istructs.PartitionID]struct {
			cfg AsyncActualizerConf
			run map[appdef.QName]struct {
				actualizer *asyncActualizer
				cancel     func()
			}
		}
	}
}

func newActualizers(cfg BasicAsyncActualizerConfig) *actualizers {
	a := &actualizers{
		cfg: cfg,
		apps: make(map[appdef.AppQName]struct {
			parts map[istructs.PartitionID]struct {
				cfg AsyncActualizerConf
				run map[appdef.QName]struct {
					actualizer *asyncActualizer
					cancel     func()
				}
			}
		}),
	}
	return a
}

func (a *actualizers) DeployPartition(appdef.AppQName, istructs.PartitionID) error { return nil }
func (a *actualizers) UndeployPartition(appdef.AppQName, istructs.PartitionID)     {}

func (a *actualizers) SetAppPartitions(appParts appparts.IAppPartitions) {
	if (appParts != nil) && (a.cfg.AppPartitions != appParts) {
		panic(fmt.Errorf("unable to reset application partitions: %w", errors.ErrUnsupported))
	}
	a.cfg.AppPartitions = appParts
}
