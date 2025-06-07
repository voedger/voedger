/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"fmt"
	"net/http"
	"net/mail"
	"strings"
)

func IsValidEmailTemplate(emailTemplate string) bool {
	if !strings.HasPrefix(emailTemplate, EmailTemplatePrefix_Text) && !strings.HasPrefix(emailTemplate, emailTemplatePrefix_Resource) {
		return false
	}
	return true
}

func TruncateEmailTemplate(emailTemplate string) string {
	if strings.HasPrefix(emailTemplate, EmailTemplatePrefix_Text) {
		return emailTemplate[len(EmailTemplatePrefix_Text):]
	}
	return emailTemplate[len(emailTemplatePrefix_Resource):]
}

func ValidateEMail(email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("email validation failed: %s", err.Error()))
	}
	return nil
}
