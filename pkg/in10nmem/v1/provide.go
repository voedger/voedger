/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 */

package in10nmemv1

import (
	"time"

	"github.com/voedger/voedger/pkg/in10n"
)

func Provide(quotas in10n.Quotas) (in10n.IN10nBroker) {
	return ProvideEx(quotas, time.Now)
}

func ProvideEx(quotas in10n.Quotas, now func() time.Time) (in10n.IN10nBroker) {
	return NewN10nBroker(quotas, now)
}
