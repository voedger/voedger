/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package ctrls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/voedger/voedger/cmd/edger/internal/states"
)

type superController struct {
	factories        map[states.AttributeKind]MicroControllerFactory
	microControllers map[string]MicroController
	currentState     states.ActualState
	params           SuperControllerParams
}

func newSuperController(factories map[states.AttributeKind]MicroControllerFactory, params SuperControllerParams) (ISuperController, error) {
	super := superController{
		factories:        factories,
		microControllers: make(map[string]MicroController),
		currentState:     states.MakeActualState(),
		params:           params,
	}
	return &super, super.loadState()
}

// ISuperController.AchieveState()
func (super *superController) AchieveState(ctx context.Context, desired states.DesiredState) (states.ActualState, error) {
	errs := make([]error, 0)

	for id, desiredAttr := range desired {
		if !states.IsScheduledTimeArrived(desiredAttr.ScheduleTime) {
			continue
		}

		currentAttr, ok := super.currentState[id]
		if !ok {
			currentAttr = states.ActualAttribute{Kind: desiredAttr.Kind}
		}
		if currentAttr.Offset != desiredAttr.Offset {
			currentAttr.AttemptNum = 1
		} else {
			if currentAttr.Error != "" {
				currentAttr.AttemptNum++
			}
		}

		if currentAttr.Achieves(desiredAttr) {
			continue // already achieved attribute
		}

		mc := super.getMicrocontroller(desiredAttr.Kind, id)
		newStatus, newInfo, err := mc(ctx, desiredAttr)

		currentAttr.Offset = desiredAttr.Offset
		currentAttr.TimeMs = time.Now().UnixMilli()
		currentAttr.Status = newStatus
		currentAttr.Info = newInfo
		if err != nil {
			currentAttr.Error = err.Error()
			errs = append(errs, fmt.Errorf(fmtAchivingStateAttributeError, desiredAttr.Kind, id, err))
		}
		super.currentState[id] = currentAttr
		if err := super.storeState(); err != nil {
			errs = append(errs, err)
		}
	}

	return super.currentState.Clone(), errors.Join(errs...)
}

// getMicrocontroller finds and returns a microcontroller with specified kind and ID. If not exists, then create it
func (super *superController) getMicrocontroller(kind states.AttributeKind, id string) MicroController {
	mc, ok := super.microControllers[id]
	if !ok {
		factory, ok := super.factories[kind]
		if !ok {
			panic(fmt.Errorf(fmtCanNotFindRegisteredFactory, kind, ErrNotFoundError))
		}
		mc = factory()
		super.microControllers[id] = mc
	}
	return mc
}

// loadState reads last achieved actual state from `edger-state.json` file
func (super *superController) loadState() error {
	fn := super.params.achievedStateFilePath()
	b, err := os.ReadFile(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf(fmtReadingSuperControllerStateFileError, fn, err)
	}

	state := states.ActualState{}
	err = json.Unmarshal(b, &state)
	if err != nil {
		return fmt.Errorf(fmtUnmarshalingStateFileError, fn, err)
	}

	super.currentState = state

	return nil
}

// loadState writes last achieved actual state to `edger-state.json` file
func (super *superController) storeState() error {
	b, err := json.Marshal(super.currentState)
	if err != nil {
		return fmt.Errorf(fmtMarshalingStateError, err)
	}

	fn := super.params.achievedStateFilePath()
	const perm = 0666
	err = os.WriteFile(fn, b, perm)
	if err != nil {
		return fmt.Errorf(fmtWriteSuperControllerStateFileError, fn, err)
	}

	return nil
}

// achievedStateFilePath returns full path and name for json-file to load and store achieved state
func (pars SuperControllerParams) achievedStateFilePath() string {
	s := pars.AchievedStateFile

	if s == "" {
		cwd, err := os.Getwd()
		if err != nil {
			panic(fmt.Errorf("can not retrieve current work directory: %w", err))
		}
		s = filepath.Join(cwd, DefaultAchievedStateFileName)
	}

	return s
}
