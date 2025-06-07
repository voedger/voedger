/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package in10nmem

import (
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/in10n"
)

func ProvideEx2(quotas in10n.Quotas, time timeu.ITime) (nb in10n.IN10nBroker, cleanup func()) {
	return NewN10nBroker(quotas, time)
}
