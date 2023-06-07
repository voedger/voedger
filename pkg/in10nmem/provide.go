/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 */

package in10nmem

import (
	"time"

	"github.com/voedger/voedger/pkg/in10n"
)

func Provide(quotas in10n.Quotas) in10n.IN10nBroker {
	return ProvideEx(quotas, time.Now)
}

func ProvideEx(quotas in10n.Quotas, now func() time.Time) (nb in10n.IN10nBroker) {
	nb, _ = NewN10nBroker(quotas, now)
	return nb
}

func ProvideEx2(quotas in10n.Quotas, now func() time.Time) (nb in10n.IN10nBroker, cleanup func()) {
	return NewN10nBroker(quotas, now)
}
