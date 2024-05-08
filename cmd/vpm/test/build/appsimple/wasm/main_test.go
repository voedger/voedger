/*
 * Copyright (c) 2024-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package main

import (
	"testing"

	"appsimple/wasm/exttest"
	"appsimple/wasm/orm"

	"github.com/voedger/voedger/pkg/exttinygo"
)

func Test_FillPbillDates(t *testing.T) {

	// Should use test name to check appdef and prepare test context
	// FillPbillDates
	exttest.InitTest(t)

	// Prepare first test data
	{
		stateIntents := []exttinygo.TIntent{
			/*
				orm.NewEvent().Set_QName(orm.Package_air.Command_Pbill.QName()).
			*/
			orm.Package_air.View_PbillDates.NewIntent(2020, 1).Set_FirstOffset(20).Set_LastOffset(17).Intent(),
			orm.Package_air.View_PbillDates.NewIntent(2021, 2).Set_FirstOffset(21).Set_LastOffset(18).Intent(),
			// exttest.Event.NewIntent().Set_QName("").Intent(),
		}
		expectedIntents := []exttinygo.TIntent{
			orm.Package_air.View_PbillDates.NewIntent(2020, 1).Set_FirstOffset(20).Set_LastOffset(17).Intent(),
			orm.Package_air.View_PbillDates.NewIntent(2021, 2).Set_FirstOffset(21).Set_LastOffset(18).Intent(),
		}

		_ = stateIntents
		_ = expectedIntents
		// exttest.RunProjectorAndCheck(event, stateIntents, Pbill, expectedIntents)

		{
			// Test intents
			{

			}
		}
	}
}
