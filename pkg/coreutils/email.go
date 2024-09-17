/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import "strings"

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
