/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
)

// # Implements:
//   - Container
type container struct {
	comment
	emb       interface{}
	name      string
	qName     QName
	typ       IType
	minOccurs Occurs
	maxOccurs Occurs
}

func newContainer(embeds interface{}, name string, typeName QName, minOccurs, maxOccurs Occurs) *container {
	return &container{
		emb:       embeds,
		name:      name,
		qName:     typeName,
		minOccurs: minOccurs,
		maxOccurs: maxOccurs,
	}
}

func (cont *container) Type() IType {
	if (cont.typ == nil) || (cont.typ.QName() != cont.QName()) {
		cont.typ = cont.embeds().App().TypeByName(cont.QName())
	}
	return cont.typ
}

func (cont *container) MaxOccurs() Occurs { return cont.maxOccurs }

func (cont *container) MinOccurs() Occurs { return cont.minOccurs }

func (cont *container) Name() string { return cont.name }

func (cont *container) QName() QName { return cont.qName }

func (cont container) String() string {
	return fmt.Sprintf("container «%s: %v»", cont.Name(), cont.QName())
}

func (cont *container) embeds() IStructure { return cont.emb.(IStructure) }

// # Implements:
//   - IContainers
//   - IContainersBuilder
type containers struct {
	emb               interface{}
	containers        map[string]*container
	containersOrdered []string
}

func makeContainers(embeds interface{}) containers {
	c := containers{embeds, make(map[string]*container), make([]string, 0)}
	return c
}

func (c *containers) AddContainer(name string, contType QName, minOccurs, maxOccurs Occurs, comment ...string) IContainersBuilder {
	if name == NullName {
		panic(fmt.Errorf("%v: empty container name: %w", c.embeds(), ErrNameMissed))
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("%v: invalid container name «%v»: %w", c.embeds(), name, err))
	}
	if c.Container(name) != nil {
		panic(fmt.Errorf("%v: container «%v» is already exists: %w", c.embeds(), name, ErrNameUniqueViolation))
	}

	if contType == NullQName {
		panic(fmt.Errorf("%v: missed container «%v» type name: %w", c.embeds(), name, ErrNameMissed))
	}

	if maxOccurs == 0 {
		panic(fmt.Errorf("%v: max occurs value (0) must be positive number: %w", c.embeds(), ErrInvalidOccurs))
	}
	if maxOccurs < minOccurs {
		panic(fmt.Errorf("%v: max occurs (%v) must be greater or equal to min occurs (%v): %w", c.embeds(), maxOccurs, minOccurs, ErrInvalidOccurs))
	}

	if typ := c.embeds().App().TypeByName(contType); typ != nil {
		if k := c.embeds().Kind(); !k.ContainerKindAvailable(typ.Kind()) {
			panic(fmt.Errorf("%v: type kind «%s» does not support child container kind «%s»: %w", c.embeds(), k.TrimString(), typ.Kind().TrimString(), ErrInvalidTypeKind))
		}
	}

	if len(c.containers) >= MaxTypeContainerCount {
		panic(fmt.Errorf("%v: maximum container count (%d) exceeds: %w", c.embeds(), MaxTypeContainerCount, ErrTooManyContainers))
	}

	cont := newContainer(c.emb, name, contType, minOccurs, maxOccurs)
	cont.SetComment(comment...)
	c.containers[name] = cont
	c.containersOrdered = append(c.containersOrdered, name)

	return c.emb.(IContainersBuilder)
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

func (c *containers) embeds() IStructure {
	return c.emb.(IStructure)
}

// Validates specified containers.
//
// # Validation:
//   - every container type must be known,
//   - every container type kind must be compatible with parent type kind
func validateTypeContainers(t IType) (err error) {
	if cnt, ok := t.(IContainers); ok {
		// resolve containers types
		cnt.Containers(func(cont IContainer) {
			contType := cont.Type()
			if contType == nil {
				err = errors.Join(err, fmt.Errorf("%v: container «%s» uses unknown type «%v»: %w", t, cont.Name(), cont.QName(), ErrNameNotFound))
				return
			}
			if !t.Kind().ContainerKindAvailable(contType.Kind()) {
				err = errors.Join(err, fmt.Errorf("%v: container «%s» type %v is incompatible: «%s» can`t contain «%s»: %w", t, cont.Name(), contType, t.Kind().TrimString(), contType.Kind().TrimString(), ErrInvalidTypeKind))
			}
		})
	}
	return err
}
