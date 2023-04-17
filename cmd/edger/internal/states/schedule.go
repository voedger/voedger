/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package states

import "time"

func isScheduledTimeArrived(scheduledTime time.Time, now time.Time) bool {
	return now.After(scheduledTime) || now.Equal(scheduledTime)
}
