/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package containers

import (
	"errors"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/internal/comments"
)

// # Supports:
//   - appdef.Container
type Container struct {
	comments.WithComments
	ws        appdef.IWorkspace
	name      string
	qName     appdef.QName
	typ       appdef.IStructure
	minOccurs appdef.Occurs
	maxOccurs appdef.Occurs
}

func NewContainer(ws appdef.IWorkspace, name string, typeName appdef.QName, minOccurs, maxOccurs appdef.Occurs) *Container {
	return &Container{
		ws:        ws,
		name:      name,
		qName:     typeName,
		minOccurs: minOccurs,
		maxOccurs: maxOccurs,
	}
}

func (cont *Container) Type() appdef.IStructure {
	if (cont.typ == nil) || (cont.typ.QName() != cont.QName()) {
		cont.typ = appdef.Structure(cont.ws.Type, cont.QName())
	}
	return cont.typ
}

func (cont Container) MaxOccurs() appdef.Occurs { return cont.maxOccurs }

func (cont Container) MinOccurs() appdef.Occurs { return cont.minOccurs }

func (cont Container) Name() string { return cont.name }

func (cont Container) QName() appdef.QName { return cont.qName }

func (cont Container) String() string {
	return fmt.Sprintf("container «%s: %v»", cont.Name(), cont.QName())
}

// # Supports:
//   - appdef.IWithContainers
type WithContainers struct {
	ws                appdef.IWorkspace
	typeKind          appdef.TypeKind
	containers        map[string]*Container
	containersOrdered []appdef.IContainer
}

func MakeWithContainers(ws appdef.IWorkspace, typeKind appdef.TypeKind) WithContainers {
	cc := WithContainers{
		ws:                ws,
		typeKind:          typeKind,
		containers:        make(map[string]*Container),
		containersOrdered: make([]appdef.IContainer, 0)}
	return cc
}

func (cc WithContainers) Container(name string) appdef.IContainer {
	if c, ok := cc.containers[name]; ok {
		return c
	}
	return nil
}

func (cc WithContainers) ContainerCount() int { return len(cc.containersOrdered) }

func (cc WithContainers) Containers() []appdef.IContainer {
	return cc.containersOrdered
}

func (cc *WithContainers) addContainer(name string, contType appdef.QName, minOccurs, maxOccurs appdef.Occurs, comment ...string) {
	if name == appdef.NullName {
		panic(appdef.ErrMissed("container name"))
	}
	if ok, err := appdef.ValidIdent(name); !ok {
		panic(fmt.Errorf("invalid container name «%v»: %w", name, err))
	}
	if cc.Container(name) != nil {
		panic(appdef.ErrAlreadyExists("container «%v»", name))
	}

	if contType == appdef.NullQName {
		panic(appdef.ErrMissed("container «%v» type", name))
	}

	if maxOccurs == 0 {
		panic(appdef.ErrOutOfBounds("max occurs value should be positive number"))
	}
	if maxOccurs < minOccurs {
		panic(appdef.ErrOutOfBounds("max occurs should be greater than or equal to min occurs (%v)", minOccurs))
	}

	if child := appdef.Structure(cc.ws.Type, contType); child != nil {
		if (cc.typeKind != appdef.TypeKind_null) && !cc.typeKind.ContainerKindAvailable(child.Kind()) {
			panic(appdef.ErrInvalid("%v can not to be a child of «%v»", child, cc.typeKind.TrimString()))
		}
	}

	if len(cc.containers) >= appdef.MaxTypeContainerCount {
		panic(appdef.ErrTooMany("containers, maximum is %d", appdef.MaxTypeContainerCount))
	}

	cont := NewContainer(cc.ws, name, contType, minOccurs, maxOccurs)
	comments.SetComment(&cont.WithComments, comment...)
	cc.containers[name] = cont
	cc.containersOrdered = append(cc.containersOrdered, cont)
}

// Validates specified containers.
//
// # Validation:
//   - every container type must be known,
//   - every container type kind must be compatible with parent type kind
func ValidateTypeContainers(t appdef.IType) (err error) {
	if cnt, ok := t.(appdef.IWithContainers); ok {
		// resolve containers types
		for _, cont := range cnt.Containers() {
			contType := cont.Type()
			if contType == nil {
				err = errors.Join(err,
					appdef.ErrNotFound("%v container «%s» type «%v»", t, cont.Name(), cont.QName()))
				continue
			}
			if !t.Kind().ContainerKindAvailable(contType.Kind()) {
				err = errors.Join(err,
					appdef.ErrInvalid("type «%v» can not to be a child of «%v»", contType, t))
			}
		}
	}
	return err
}

// # Supports:
//   - appdef.IContainersBuilder
type ContainersBuilder struct {
	cc *WithContainers
}

func MakeContainersBuilder(cc *WithContainers) ContainersBuilder {
	return ContainersBuilder{cc}
}

func (cb *ContainersBuilder) AddContainer(name string, typeName appdef.QName, minimum, maximum appdef.Occurs, comment ...string) appdef.IContainersBuilder {
	cb.cc.addContainer(name, typeName, minimum, maximum, comment...)
	return cb
}
