/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package edger

import "time"

type EdgerParams struct {
	// AchievedStateFilePath is file name to load and store last achieved state.
	// Ref. to mctrls.SuperControllerParams.AchievedStateFilePath field
	AchievedStateFilePath string

	// AchieveAttemptInterval is time interval between achieving attempts if first attempt has finished with errors.
	// Ref. to superControllerCycle() method
	AchieveAttemptInterval time.Duration
}
