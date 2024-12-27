/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package blobprocessor

import (
	"log"
	"testing"
)

type base struct {
	fld1 int32
	fld2 int32
	fld3 int32
}

type Intf1 interface {
	Fld1() int32
}

type Intf2 interface {
	Intf1
	Fld2() int32
}
type Intf3 interface {
	Fld3() int32
}

func (b base) Fld1() int32 {
	return b.fld1
}

func (b base) Fld2() int32 {
	return b.fld2
}
func (b base) Fld3() int32 {
	return b.fld3
}

func NewIntf1() Intf1 {
	return &base{}
}

func NewIntf2() Intf2 {
	return &base{}
}

func NewIntf3() Intf3 {
	return &base{}
}

func TestBase(t *testing.T) {
	intf1 := NewIntf1()
	switch intf1.(type) {
	case Intf3:
		log.Println(3)
	case Intf2:
		log.Println(2)
	case Intf1:
		log.Println(1)
	}
}
