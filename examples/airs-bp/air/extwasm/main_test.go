package main

import (
	"extwasm/exttest"
	"extwasm/orm"
	"testing"

	exttinygo "github.com/voedger/exttinygo"
)

func Test_Pbill(t *testing.T) {

	// // Should use test name to check appdef and prepare test context
	exttest.InitTest(t)

	// Prepare first test data
	{
		stateIntents := []exttinygo.TIntent{
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
		// exttest.RunAndCheck(stateIntents, Pbill, expectedIntents)

		{
			// Test intents
			{

			}
		}
	}

	// Prepare second test data
	{
		exttest.ResetState()
		{
			intent := orm.Package_air.View_PbillDates.NewIntent(3020, 2)
			intent.Set_FirstOffset(20)
			intent.Set_LastOffset(17)

			exttest.CommitIntentsToState()
		}

		MyProjector()
		// Test intents
		{

		}
	}

}
