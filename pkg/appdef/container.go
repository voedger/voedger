/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"strings"
)

const SystemContainer_ViewPartitionKey = SystemPackagePrefix + "pkey"
const SystemContainer_ViewClusteringCols = SystemPackagePrefix + "ccols"
const SystemContainer_ViewKey = SystemPackagePrefix + "key"
const SystemContainer_ViewValue = SystemPackagePrefix + "val"

// # Implements:
//   - Container
type container struct {
	parent    interface{}
	name      string
	qName     QName
	def       IDef
	minOccurs Occurs
	maxOccurs Occurs
}

func newContainer(parent interface{}, name string, def QName, minOccurs, maxOccurs Occurs) *container {
	return &container{
		parent:    parent,
		name:      name,
		qName:     def,
		minOccurs: minOccurs,
		maxOccurs: maxOccurs,
	}
}

func (cont *container) Def() IDef {
	if (cont.def == nil) || (cont.def.QName() != cont.QName()) {
		cont.def = cont.parentDef().App().DefByName(cont.QName())
	}
	return cont.def
}

func (cont *container) IsSys() bool { return IsSysContainer(cont.name) }

func (cont *container) MaxOccurs() Occurs { return cont.maxOccurs }

func (cont *container) MinOccurs() Occurs { return cont.minOccurs }

func (cont *container) Name() string { return cont.name }

func (cont *container) QName() QName { return cont.qName }

func (cont *container) parentDef() IDef {
	return cont.parent.(IDef)
}

// Returns is container system
func IsSysContainer(n string) bool {
	return strings.HasPrefix(n, SystemPackagePrefix) && // fast check
		// then more accuracy
		((n == SystemContainer_ViewPartitionKey) ||
			(n == SystemContainer_ViewClusteringCols) ||
			(n == SystemContainer_ViewKey) ||
			(n == SystemContainer_ViewValue))
}

// # Implements:
//   - IContainers
//   - IContainersBuilder
type containers struct {
	parent            interface{}
	containers        map[string]*container
	containersOrdered []string
}

func makeContainers(def interface{}) containers {
	c := containers{def, make(map[string]*container), make([]string, 0)}
	return c
}

func (c *containers) AddContainer(name string, contDef QName, minOccurs, maxOccurs Occurs) IContainersBuilder {
	if name == NullName {
		panic(fmt.Errorf("%v: empty container name: %w", c.parentDef().QName(), ErrNameMissed))
	}
	if !IsSysContainer(name) {
		if ok, err := ValidIdent(name); !ok {
			panic(fmt.Errorf("%v: invalid container name «%v»: %w", c.parentDef().QName(), name, err))
		}
	}
	if c.Container(name) != nil {
		panic(fmt.Errorf("%v: container «%v» is already exists: %w", c.parentDef().QName(), name, ErrNameUniqueViolation))
	}

	if contDef == NullQName {
		panic(fmt.Errorf("%v: missed container «%v» definition name: %w", c.parentDef().QName(), name, ErrNameMissed))
	}

	if maxOccurs == 0 {
		panic(fmt.Errorf("%v: max occurs value (0) must be positive number: %w", c.parentDef().QName(), ErrInvalidOccurs))
	}
	if maxOccurs < minOccurs {
		panic(fmt.Errorf("%v: max occurs (%v) must be greater or equal to min occurs (%v): %w", c.parentDef().QName(), maxOccurs, minOccurs, ErrInvalidOccurs))
	}

	if cd := c.parentDef().App().DefByName(contDef); cd != nil {
		if k := c.parentDef().Kind(); !k.ContainerKindAvailable(cd.Kind()) {
			panic(fmt.Errorf("%v: definition kind «%s» does not support child container kind «%s»: %w", c.parentDef().QName(), k.TrimString(), cd.Kind().TrimString(), ErrInvalidDefKind))
		}
	}

	if len(c.containers) >= MaxDefContainerCount {
		panic(fmt.Errorf("%v: maximum container count (%d) exceeds: %w", c.parentDef().QName(), MaxDefContainerCount, ErrTooManyContainers))
	}

	cont := newContainer(c.parent, name, contDef, minOccurs, maxOccurs)
	c.containers[name] = cont
	c.containersOrdered = append(c.containersOrdered, name)

	return c.parent.(IContainersBuilder)
}

func (c *containers) Container(name string) IContainer {
	if c, ok := c.containers[name]; ok {
		return c
	}
	return nil
}

func (c *containers) ContainerCount() int {
	return len(c.containersOrdered)
}

func (c *containers) Containers(cb func(IContainer)) {
	for _, n := range c.containersOrdered {
		cb(c.Container(n))
	}
}

func (c *containers) parentDef() IDef {
	return c.parent.(IDef)
}

// Validates specified containers.
//
// # Validation:
//   - every container definition must be known,
//   - every container definition kind must be compatible with parent definition kind
func validateDefContainers(def IDef) (err error) {
	if cnt, ok := def.(IContainers); ok {
		// resolve containers definitions
		cnt.Containers(func(cont IContainer) {
			contDef := cont.Def()
			if contDef == nil {
				err = errors.Join(err, fmt.Errorf("%v: container «%s» uses unknown definition «%v»: %w", def.QName(), cont.Name(), cont.QName(), ErrNameNotFound))
				return
			}
			if !def.Kind().ContainerKindAvailable(contDef.Kind()) {
				err = errors.Join(err, fmt.Errorf("%v: container «%s» definition «%v» is incompatible: «%s» can`t contain «%s»: %w", def.QName(), cont.Name(), cont.QName(), def.Kind().TrimString(), contDef.Kind().TrimString(), ErrInvalidDefKind))
			}
		})
	}
	return err
}
