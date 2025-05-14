/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author Aleksei Ponomarev
 *
 */

package itokensjwt

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/goutils/timeu"
	"github.com/voedger/voedger/pkg/istructs"

	"github.com/golang-jwt/jwt/v5"

	"github.com/voedger/voedger/pkg/itokens"
)

var (
	onByteArrayMutate       func(array *[]byte) // used in tests
	onSecretKeyMutate       func() interface{}
	onTokenPartsMutate      func(str []string) string
	onTokenArrayPartsMutate func() []string
)

func (j *JWTSigner) IssueToken(app appdef.AppQName, duration time.Duration, pointerToPayload interface{}) (token string, err error) {
	//	var duration float64
	var b []byte
	audience := reflect.TypeOf(pointerToPayload).Elem()
	m := make(map[string]interface{})
	b, err = json.Marshal(pointerToPayload)
	if err != nil {
		err = fmt.Errorf("cannot marshal input payload %w", itokens.ErrInvalidPayload)
		return "", err
	}
	if onByteArrayMutate != nil {
		onByteArrayMutate(&b)
	}
	err = json.Unmarshal(b, &m)
	if err != nil {
		err = fmt.Errorf("error unmarshal token to mapClaims %w", itokens.ErrInvalidPayload)
		return "", err
	}

	now := j.iTime.Now()
	claims := &jwt.MapClaims{
		"iat":      now.Unix(),
		"exp":      now.Add(duration).Unix(),
		"aud":      audience.String(),
		"Duration": duration,
		"AppQName": &app,
		"IssuedAt": now,
	}
	*claims = mergeClaimsMaps(m, *claims)
	token, err = j.sign(claims)
	if err != nil {
		err = fmt.Errorf("cannot issue token %w", itokens.ErrSignerError)
		return "", err
	}
	return token, err
}

func (j *JWTSigner) ValidateToken(token string, pointerToPayload interface{}) (gp istructs.GenericPayload, err error) {
	var (
		audience string
		jwtToken *jwt.Token
	)
	expectedAudience := reflect.TypeOf(pointerToPayload).Elem().String()
	parser := jwt.NewParser(jwt.WithJSONNumber(), jwt.WithTimeFunc(j.iTime.Now))
	jwtToken, err = parser.Parse(token, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, itokens.ErrInvalidToken
		}
		return j.secretKey, nil
	})
	if jwtToken == nil {
		if err != nil {
			err = fmt.Errorf(err.Error()+". %w", itokens.ErrInvalidToken)
		}
		return gp, err
	}
	if jwtToken.Claims != nil {
		audience = jwtToken.Claims.(jwt.MapClaims)["aud"].(string)

		p, errPayload := buildGenericPayload(jwtToken.Claims)
		if errPayload != nil {
			err = fmt.Errorf("cannot build generic payload %w", itokens.ErrInvalidPayload)
			return gp, err
		}
		gp = p
	}

	if jwtToken.Valid {
		if strings.Compare(expectedAudience, audience) != 0 {
			return gp, fmt.Errorf(errorVerifyAudience, expectedAudience,
				audience, itokens.ErrInvalidAudience)
		}
		parts := strings.Split(token, ".")
		if onTokenArrayPartsMutate != nil {
			parts = onTokenArrayPartsMutate()
		}
		if len(parts) != numberOfParts {
			err = fmt.Errorf("error split raw token, token is malformed. %w", itokens.ErrInvalidPayload)
			return gp, err
		}
		var (
			claimBytes    []byte
			payloadClaims string
		)
		payloadClaims = getTokenPayload(parts)
		if onTokenPartsMutate != nil {
			payloadClaims = onTokenPartsMutate(parts)
		}
		if claimBytes, err = base64.RawURLEncoding.DecodeString(payloadClaims); err != nil {
			err = fmt.Errorf("error decode the claims part of token %w", err)
			return gp, err
		}
		if onByteArrayMutate != nil {
			onByteArrayMutate(&claimBytes)
		}

		err = coreutils.JSONUnmarshal(claimBytes, &pointerToPayload)

		return gp, err
	}
	err = setErrorDescription(err)
	return gp, err
}

func buildGenericPayload(claims jwt.Claims) (gp istructs.GenericPayload, err error) {
	var (
		duration int64
		issuedAt time.Time
	)
	duration, err = claims.(jwt.MapClaims)["Duration"].(json.Number).Int64()
	if err != nil {
		return gp, err
	}

	iat, e := json.Marshal(claims.(jwt.MapClaims)["IssuedAt"])
	if e != nil {
		err = fmt.Errorf("cannot marshal input payload %w", itokens.ErrInvalidPayload)
		return gp, err
	}
	e = json.Unmarshal(iat, &issuedAt)
	if e != nil {
		err = fmt.Errorf("cannot unmarshal input payload %w", itokens.ErrInvalidPayload)
		return gp, err
	}

	b, e := json.Marshal(claims.(jwt.MapClaims)["AppQName"])
	if e != nil {
		err = fmt.Errorf("cannot marshal input payload %w", itokens.ErrInvalidPayload)
		return gp, err
	}
	qname := appdef.AppQName{}
	e = json.Unmarshal(b, &qname)
	if e != nil {
		err = fmt.Errorf("error unmarshal token to mapClaims %w", itokens.ErrInvalidPayload)
		return gp, err
	}
	gp.AppQName = qname
	gp.Duration = time.Duration(duration)
	gp.IssuedAt = issuedAt
	return gp, err
}

func (j *JWTSigner) CryptoHash256(data []byte) (hash [hashLength]byte) {
	hs256 := jwt.SigningMethodHS256
	hasher := hmac.New(hs256.Hash.New, j.secretKey)
	hasher.Write(data)
	expectedHash := hasher.Sum(nil)
	copy(hash[:], expectedHash)
	return
}

//nolint:errorlint
func setErrorDescription(err error) error {
	if errors.Is(err, jwt.ErrTokenExpired) {
		err = itokens.ErrTokenExpired
	}
	if errors.Is(err, jwt.ErrTokenUnverifiable) {
		err = itokens.ErrInvalidToken
	}
	return err
}

func (j *JWTSigner) sign(claims jwt.Claims) (token string, err error) {

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if onSecretKeyMutate != nil {
		return jwtToken.SignedString(onSecretKeyMutate())
	}
	return jwtToken.SignedString(j.secretKey)
}

func getTokenPayload(token []string) string {
	return token[1]
}

func NewJWTSigner(secretKey SecretKeyType, iTime timeu.ITime) *JWTSigner {
	var byteSecretKey []byte = secretKey
	if len(byteSecretKey) < SecretKeyLength {
		panic(fmt.Errorf("invalid key length: must be %d chars", SecretKeyLength))
	}
	return &JWTSigner{byteSecretKey, iTime}
}

func mergeClaimsMaps(maps ...map[string]interface{}) (result map[string]interface{}) {
	result = make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
