/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// # Implements:
//  - IWithAbstract
type withAbstract struct {
	abstract bool
}

func makeWithAbstract() withAbstract {
	return withAbstract{}
}

func (a *withAbstract) Abstract() bool { return a.abstract }

func (a *withAbstract) setAbstract() { a.abstract = true }

// # Implements:
//  - IWithAbstractBuilder
type withAbstractBuilder struct {
	*withAbstract
}

func makeWithAbstractBuilder(a *withAbstract) withAbstractBuilder {
	return withAbstractBuilder{a}
}

func (ab *withAbstractBuilder) SetAbstract() { ab.setAbstract() }
