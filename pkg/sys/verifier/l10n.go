/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package verifier

import (
	"golang.org/x/text/language"

	coreutils "github.com/voedger/voedger/pkg/utils"
)

var translations = coreutils.Translations{
	"Here is your verification code": {
		language.English: "Here is your verification code",
		language.German:  "Bij deze je verificatiecode",
		language.French:  "Voici votre code de vérification",
	},
	"Please, enter this code on": {
		language.English: "Please, enter this code on",
		language.German:  "Vul deze verificatiecode in op",
		language.French:  "Veuillez entrer ce code sur",
	},
	"to confirm your email.": {
		language.English: "to confirm your email.",
		language.German:  "om je mailadres te bevestigen.",
		language.French:  "pour confirmer votre e-mail.",
	},
	"Your verification code": {
		language.English: "Your verification code",
		language.German:  "Verificatiecode",
		language.French:  "Votre code de vérification",
	},
}
