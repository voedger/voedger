/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

func newContainer(name string, schema QName, minOccurs, maxOccurs Occurs) Container {
	return Container{
		name:      name,
		schema:    schema,
		minOccurs: minOccurs,
		maxOccurs: maxOccurs,
	}
}

// Returns is container system
func (cont *Container) IsSys() bool { return IsSysContainer(cont.name) }

// —————————— istructs.IContainerDesc ——————————

func (cont *Container) Name() string { return cont.name }

func (cont *Container) Schema() QName { return cont.schema }

func (cont *Container) MinOccurs() Occurs { return cont.minOccurs }

func (cont *Container) MaxOccurs() Occurs { return cont.maxOccurs }
