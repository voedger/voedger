/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"strconv"
)

// # Implements:
//   - Container
type container struct {
	comment
	app       *appDef
	name      string
	qName     QName
	typ       IType
	minOccurs Occurs
	maxOccurs Occurs
}

func newContainer(app *appDef, name string, typeName QName, minOccurs, maxOccurs Occurs) *container {
	return &container{
		app:       app,
		name:      name,
		qName:     typeName,
		minOccurs: minOccurs,
		maxOccurs: maxOccurs,
	}
}

func (cont *container) Type() IType {
	if (cont.typ == nil) || (cont.typ.QName() != cont.QName()) {
		cont.typ = cont.app.TypeByName(cont.QName())
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

// # Implements:
//   - IContainers
type containers struct {
	app               *appDef
	typeKind          TypeKind
	containers        map[string]*container
	containersOrdered []IContainer
}

func makeContainers(app *appDef, typeKind TypeKind) containers {
	c := containers{
		app:               app,
		typeKind:          typeKind,
		containers:        make(map[string]*container),
		containersOrdered: make([]IContainer, 0)}
	return c
}

func (c containers) Container(name string) IContainer {
	if c, ok := c.containers[name]; ok {
		return c
	}
	return nil
}

func (c containers) ContainerCount() int {
	return len(c.containersOrdered)
}

func (c containers) Containers() []IContainer {
	return c.containersOrdered
}

func (cc *containers) addContainer(name string, contType QName, minOccurs, maxOccurs Occurs, comment ...string) {
	if name == NullName {
		panic(fmt.Errorf("empty container name: %w", ErrNameMissed))
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("invalid container name «%v»: %w", name, err))
	}
	if cc.Container(name) != nil {
		panic(fmt.Errorf("container «%v» is already exists: %w", name, ErrNameUniqueViolation))
	}

	if contType == NullQName {
		panic(fmt.Errorf("missed container «%v» type name: %w", name, ErrNameMissed))
	}

	if maxOccurs == 0 {
		panic(fmt.Errorf("max occurs value (0) must be positive number: %w", ErrInvalidOccurs))
	}
	if maxOccurs < minOccurs {
		panic(fmt.Errorf("max occurs (%v) must be greater or equal to min occurs (%v): %w", maxOccurs, minOccurs, ErrInvalidOccurs))
	}

	if typ := cc.app.TypeByName(contType); typ != nil {
		if (cc.typeKind != TypeKind_null) && !cc.typeKind.ContainerKindAvailable(typ.Kind()) {
			panic(fmt.Errorf("type kind «%s» does not support child container kind «%s»: %w", cc.typeKind.TrimString(), typ.Kind().TrimString(), ErrInvalidTypeKind))
		}
	}

	if len(cc.containers) >= MaxTypeContainerCount {
		panic(fmt.Errorf("maximum container count (%d) exceeds: %w", MaxTypeContainerCount, ErrTooManyContainers))
	}

	cont := newContainer(cc.app, name, contType, minOccurs, maxOccurs)
	cont.comment.setComment(comment...)
	cc.containers[name] = cont
	cc.containersOrdered = append(cc.containersOrdered, cont)
}

// Validates specified containers.
//
// # Validation:
//   - every container type must be known,
//   - every container type kind must be compatible with parent type kind
func validateTypeContainers(t IType) (err error) {
	if cnt, ok := t.(IContainers); ok {
		// resolve containers types
		for _, cont := range cnt.Containers() {
			contType := cont.Type()
			if contType == nil {
				err = errors.Join(err, fmt.Errorf("%v: container «%s» uses unknown type «%v»: %w", t, cont.Name(), cont.QName(), ErrNameNotFound))
				continue
			}
			if !t.Kind().ContainerKindAvailable(contType.Kind()) {
				err = errors.Join(err, fmt.Errorf("%v: container «%s» type %v is incompatible: «%s» can`t contain «%s»: %w", t, cont.Name(), contType, t.Kind().TrimString(), contType.Kind().TrimString(), ErrInvalidTypeKind))
			}
		}
	}
	return err
}

// # Implements:
//   - IContainersBuilder
type containersBuilder struct {
	*containers
}

func makeContainersBuilder(containers *containers) containersBuilder {
	return containersBuilder{
		containers: containers,
	}
}

func (cb *containersBuilder) AddContainer(name string, typeName QName, min, max Occurs, comment ...string) IContainersBuilder {
	cb.addContainer(name, typeName, min, max, comment...)
	return cb
}

func (o Occurs) String() string {
	switch o {
	case Occurs_Unbounded:
		return Occurs_UnboundedStr
	default:
		const base = 10
		return strconv.FormatUint(uint64(o), base)
	}
}

func (o Occurs) MarshalJSON() ([]byte, error) {
	s := o.String()
	if o == Occurs_Unbounded {
		s = strconv.Quote(s)
	}
	return []byte(s), nil
}

func (o *Occurs) UnmarshalJSON(data []byte) (err error) {
	switch string(data) {
	case strconv.Quote(Occurs_UnboundedStr):
		*o = Occurs_Unbounded
		return nil
	default:
		var i uint64
		const base, wordBits = 10, 16
		i, err = strconv.ParseUint(string(data), base, wordBits)
		if err == nil {
			*o = Occurs(i)
		}
		return err
	}
}
