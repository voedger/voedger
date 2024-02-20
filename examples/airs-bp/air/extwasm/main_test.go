package main

import (
	"extwasm/exttest"
	"extwasm/orm"
	"testing"
)

func Test_Projector_MyProjector(t *testing.T) {

	// Should use test name to check appdef and prepare test context
	exttest.InitTest(t)

	// Prepare first test data
	{
		{
			intent := orm.Package_air.View_PbillDates.NewIntent(2020, 1)
			intent.Set_FirstOffset(20)
			intent.Set_LastOffset(17)

			exttest.CommitIntentsToState()
		}

		MyProjector()
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
