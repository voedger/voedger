/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package iblobstorage

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
)

func NewSUUID() SUUID {
	// Generate UUID (Version 4)
	uuidPart := uuid.New()

	// Generate additional random bytes (16 bytes for 128 bits)
	randomPart := make([]byte, SUUIDRandomPartLen)
	if _, err := rand.Read(randomPart); err != nil {
		// notest
		panic(err)
	}

	// Concatenate UUID and random bytes
	combined := append(uuidPart[:], randomPart...)

	// Encode to Base64 using URL-safe encoding
	return SUUID(base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(combined))
}

func (dt DurationType) Seconds() int {
	return int(dt) * secondsInDay
}
