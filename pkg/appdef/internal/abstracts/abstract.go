/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package abstracts

// # Supports:
//  - IWithAbstract
type WithAbstract struct {
	abstract bool
}

func MakeWithAbstract() WithAbstract {
	return WithAbstract{}
}

func (a *WithAbstract) Abstract() bool { return a.abstract }

func (a *WithAbstract) setAbstract() { a.abstract = true }

// # Supports:
//  - IWithAbstractBuilder
type WithAbstractBuilder struct {
	*WithAbstract
}

func MakeWithAbstractBuilder(a *WithAbstract) WithAbstractBuilder {
	return WithAbstractBuilder{a}
}

func (ab *WithAbstractBuilder) SetAbstract() { ab.setAbstract() }
