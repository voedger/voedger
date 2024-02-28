/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//  - IWithAbstract
//	- IWithAbstractBuilder
type withAbstract struct {
	abstract bool
}

func (a *withAbstract) Abstract() bool { return a.abstract }
func (a *withAbstract) SetAbstract()   { a.abstract = true }
