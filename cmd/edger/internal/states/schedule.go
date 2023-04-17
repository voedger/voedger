/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package states

import "time"

func isScheduledTimeArrived(scheduledTime time.Time, now time.Time) bool {
	return now.After(scheduledTime) || now.Equal(scheduledTime)
}
