/*
 * Copyright (c) 2025-present unTill Software Development Group B. V.
 * @author Maxim Geraskin
 */

package iextsse

import (
	"fmt"
)

// STSKey struct implements the ISTSKey interface.
type STSKey struct {
	namespace string
	name      string
	data      map[string]interface{}
}

// NewSTSKey constructs a new STSKey.
func NewSTSKey(namespace, name string) *STSKey {
	return &STSKey{
		namespace: namespace,
		name:      name,
		data:      make(map[string]interface{}),
	}
}

// Namespace returns the namespace of the key.
func (s *STSKey) Namespace() string {
	return s.namespace
}

// Name returns the name of the key.
func (s *STSKey) Name() string {
	return s.name
}

// AsInt64 retrieves an int64 value by name, panicking if not found or wrong type.
func (s *STSKey) AsInt64(name string) int64 {
	v, ok := s.data[name]
	if !ok {
		panic(fmt.Sprintf("value missing for key '%s'", name))
	}
	val, ok := v.(int64)
	if !ok {
		panic(fmt.Sprintf("value for key '%s' is not int64", name))
	}
	return val
}

// AsString retrieves a string value by name, panicking if not found or wrong type.
func (s *STSKey) AsString(name string) string {
	v, ok := s.data[name]
	if !ok {
		panic(fmt.Sprintf("value missing for key '%s'", name))
	}
	val, ok := v.(string)
	if !ok {
		panic(fmt.Sprintf("value for key '%s' is not string", name))
	}
	return val
}

// SetInt64 sets an int64 value by name.
func (s *STSKey) SetInt64(name string, value int64) {
	s.data[name] = value
}

// SetString sets a string value by name.
func (s *STSKey) SetString(name string, value string) {
	s.data[name] = value
}
