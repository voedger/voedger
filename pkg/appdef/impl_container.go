/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

// TODO: type ContainerName = string

// # Implements:
//   - Container
type container struct {
	comment
	app       *appDef
	name      string
	qName     QName
	typ       IStructure
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

func (cont *container) Type() IStructure {
	if (cont.typ == nil) || (cont.typ.QName() != cont.QName()) {
		cont.typ = Structure(cont.app.Type, cont.QName())
	}
	return cont.typ
}

func (cont container) MaxOccurs() Occurs { return cont.maxOccurs }

func (cont container) MinOccurs() Occurs { return cont.minOccurs }

func (cont container) Name() string { return cont.name }

func (cont container) QName() QName { return cont.qName }

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
	cc := containers{
		app:               app,
		typeKind:          typeKind,
		containers:        make(map[string]*container),
		containersOrdered: make([]IContainer, 0)}
	return cc
}

func (cc containers) Container(name string) IContainer {
	if c, ok := cc.containers[name]; ok {
		return c
	}
	return nil
}

func (cc containers) ContainerCount() int {
	return len(cc.containersOrdered)
}

func (cc containers) Containers() []IContainer {
	return cc.containersOrdered
}

func (cc *containers) addContainer(name string, contType QName, minOccurs, maxOccurs Occurs, comment ...string) {
	if name == NullName {
		panic(ErrMissed("container name"))
	}
	if ok, err := ValidIdent(name); !ok {
		panic(fmt.Errorf("invalid container name «%v»: %w", name, err))
	}
	if cc.Container(name) != nil {
		panic(ErrAlreadyExists("container «%v»", name))
	}

	if contType == NullQName {
		panic(ErrMissed("container «%v» type", name))
	}

	if maxOccurs == 0 {
		panic(ErrOutOfBounds("max occurs value should be positive number"))
	}
	if maxOccurs < minOccurs {
		panic(ErrOutOfBounds("max occurs should be greater than or equal to min occurs (%v)", minOccurs))
	}

	if child := Structure(cc.app.Type, contType); child != nil {
		if (cc.typeKind != TypeKind_null) && !cc.typeKind.ContainerKindAvailable(child.Kind()) {
			panic(ErrInvalid("%v can not to be a child of «%v»", child, cc.typeKind.TrimString()))
		}
	}

	if len(cc.containers) >= MaxTypeContainerCount {
		panic(ErrTooMany("containers, maximum is %d", MaxTypeContainerCount))
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
				err = errors.Join(err,
					ErrNotFound("%v container «%s» type «%v»", t, cont.Name(), cont.QName()))
				continue
			}
			if !t.Kind().ContainerKindAvailable(contType.Kind()) {
				err = errors.Join(err,
					ErrInvalid("type «%v» can not to be a child of «%v»", contType, t))
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

func (cb *containersBuilder) AddContainer(name string, typeName QName, minimum, maximum Occurs, comment ...string) IContainersBuilder {
	cb.addContainer(name, typeName, minimum, maximum, comment...)
	return cb
}

func (o Occurs) String() string {
	switch o {
	case Occurs_Unbounded:
		return Occurs_UnboundedStr
	default:
		return utils.UintToString(o)
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
