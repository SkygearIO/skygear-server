package passwordpolicy

import (
	"regexp"
	"strings"

	"github.com/nbutton23/zxcvbn-go"

	"github.com/skygeario/skygear-server/pkg/core/auth/passwordhistory"
	corepassword "github.com/skygeario/skygear-server/pkg/core/password"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
)

func isUpperRune(r rune) bool {
	// NOTE: Intentionally not use unicode.IsUpper
	// because it take other languages into account.
	return r >= 'A' && r <= 'Z'
}

func isLowerRune(r rune) bool {
	// NOTE: Intentionally not use unicode.IsLower
	// because it take other languages into account.
	return r >= 'a' && r <= 'z'
}

func isDigitRune(r rune) bool {
	// NOTE: Intentionally not use unicode.IsDigit
	// because it take other languages into account.
	return r >= '0' && r <= '9'
}

func isSymbolRune(r rune) bool {
	// We define symbol as non-alphanumeric character
	return !isUpperRune(r) && !isLowerRune(r) && !isDigitRune(r)
}

func checkPasswordLength(password string, minLength int) bool {
	if minLength <= 0 {
		return true
	}
	// There exist many ways to define the length of a string
	// For example:
	// 1. The number of bytes of a given encoding
	// 2. The number of code points
	// 3. The number of extended grapheme cluster
	// Here we use the simpliest one:
	// the number of bytes of the given string in UTF-8 encoding
	return len(password) >= minLength
}

func checkPasswordUppercase(password string) bool {
	for _, r := range password {
		if isUpperRune(r) {
			return true
		}
	}
	return false
}

func checkPasswordLowercase(password string) bool {
	for _, r := range password {
		if isLowerRune(r) {
			return true
		}
	}
	return false
}

func checkPasswordDigit(password string) bool {
	for _, r := range password {
		if isDigitRune(r) {
			return true
		}
	}
	return false
}

func checkPasswordSymbol(password string) bool {
	for _, r := range password {
		if isSymbolRune(r) {
			return true
		}
	}
	return false
}

func checkPasswordExcludedKeywords(password string, keywords []string) bool {
	if len(keywords) <= 0 {
		return true
	}
	words := []string{}
	for _, w := range keywords {
		words = append(words, regexp.QuoteMeta(w))
	}
	re, err := regexp.Compile("(?i)" + strings.Join(words, "|"))
	if err != nil {
		return false
	}
	loc := re.FindStringIndex(password)
	if loc == nil {
		return true
	}
	return false
}

func checkPasswordGuessableLevel(password string, minLevel int, userInputs []string) (int, bool) {
	if minLevel <= 0 {
		return 0, true
	}
	minScore := minLevel - 1
	if minScore > 4 {
		minScore = 4
	}
	result := zxcvbn.PasswordStrength(password, userInputs)
	ok := result.Score >= minScore
	return result.Score + 1, ok
}

func userDataToStringStringMap(m map[string]interface{}) map[string]string {
	output := make(map[string]string)
	for key, value := range m {
		str, ok := value.(string)
		if ok {
			output[key] = str
		}
	}
	return output
}

func filterDictionary(m map[string]string, predicate func(string) bool) []string {
	output := []string{}
	for key, value := range m {
		ok := predicate(key)
		if ok {
			output = append(output, value)
		}
	}
	return output
}

func filterDictionaryByKeys(m map[string]string, keys []string) []string {
	lookupMap := make(map[string]bool)
	for _, key := range keys {
		lookupMap[key] = true
	}
	predicate := func(key string) bool {
		_, ok := lookupMap[key]
		return ok
	}

	return filterDictionary(m, predicate)
}

func filterDictionaryTakeAll(m map[string]string) []string {
	predicate := func(key string) bool {
		return true
	}
	return filterDictionary(m, predicate)
}

type ValidatePasswordPayload struct {
	AuthID        string
	PlainPassword string
	UserData      map[string]interface{}
}

type PasswordChecker struct {
	PwMinLength            int
	PwUppercaseRequired    bool
	PwLowercaseRequired    bool
	PwDigitRequired        bool
	PwSymbolRequired       bool
	PwMinGuessableLevel    int
	PwExcludedKeywords     []string
	PwExcludedFields       []string
	PwHistorySize          int
	PwHistoryDays          int
	PasswordHistoryEnabled bool
	PasswordHistoryStore   passwordhistory.Store
}

func (pc *PasswordChecker) checkPasswordLength(password string) *PasswordViolation {
	minLength := pc.PwMinLength
	if minLength > 0 && !checkPasswordLength(password, minLength) {
		return &PasswordViolation{
			Reason: PasswordTooShort,
			Info: map[string]interface{}{
				"min_length": minLength,
				"pw_length":  len(password),
			},
		}
	}
	return nil
}

func (pc *PasswordChecker) checkPasswordUppercase(password string) *PasswordViolation {
	if pc.PwUppercaseRequired && !checkPasswordUppercase(password) {
		return &PasswordViolation{Reason: PasswordUppercaseRequired}
	}
	return nil
}

func (pc *PasswordChecker) checkPasswordLowercase(password string) *PasswordViolation {
	if pc.PwLowercaseRequired && !checkPasswordLowercase(password) {
		return &PasswordViolation{Reason: PasswordLowercaseRequired}
	}
	return nil
}

func (pc *PasswordChecker) checkPasswordDigit(password string) *PasswordViolation {
	if pc.PwDigitRequired && !checkPasswordDigit(password) {
		return &PasswordViolation{Reason: PasswordDigitRequired}
	}
	return nil
}

func (pc *PasswordChecker) checkPasswordSymbol(password string) *PasswordViolation {
	if pc.PwSymbolRequired && !checkPasswordSymbol(password) {
		return &PasswordViolation{Reason: PasswordSymbolRequired}
	}
	return nil
}

func (pc *PasswordChecker) checkPasswordExcludedKeywords(password string) *PasswordViolation {
	keywords := pc.PwExcludedKeywords
	if len(keywords) > 0 && !checkPasswordExcludedKeywords(password, keywords) {
		return &PasswordViolation{Reason: PasswordContainingExcludedKeywords}
	}
	return nil
}

func (pc *PasswordChecker) checkPasswordExcludedFields(password string, userData map[string]interface{}) *PasswordViolation {
	fields := pc.PwExcludedFields
	if len(fields) > 0 {
		dict := userDataToStringStringMap(userData)
		keywords := filterDictionaryByKeys(dict, fields)
		if !checkPasswordExcludedKeywords(password, keywords) {
			return &PasswordViolation{Reason: PasswordContainingExcludedKeywords}
		}
	}
	return nil
}

func (pc *PasswordChecker) checkPasswordGuessableLevel(password string, userData map[string]interface{}) *PasswordViolation {
	minLevel := pc.PwMinGuessableLevel
	if minLevel > 0 {
		dict := userDataToStringStringMap(userData)
		userInputs := filterDictionaryTakeAll(dict)
		level, ok := checkPasswordGuessableLevel(password, minLevel, userInputs)
		if !ok {
			return &PasswordViolation{
				Reason: PasswordBelowGuessableLevel,
				Info: map[string]interface{}{
					"min_level": minLevel,
					"pw_level":  level,
				},
			}
		}
	}
	return nil
}

func (pc *PasswordChecker) checkPasswordHistory(password, authID string) *PasswordViolation {
	makeErr := func() *PasswordViolation {
		return &PasswordViolation{
			Reason: PasswordReused,
			Info: map[string]interface{}{
				"history_size": pc.PwHistorySize,
				"history_days": pc.PwHistoryDays,
			},
		}
	}

	if pc.shouldCheckPasswordHistory() && authID != "" {
		history, err := pc.PasswordHistoryStore.GetPasswordHistory(
			authID,
			pc.PwHistorySize,
			pc.PwHistoryDays,
		)
		if err != nil {
			return makeErr()
		}
		for _, ph := range history {
			if IsSamePassword(ph.HashedPassword, password) {
				return makeErr()
			}
		}
	}
	return nil
}

func (pc *PasswordChecker) ValidatePassword(payload ValidatePasswordPayload) error {
	password := payload.PlainPassword
	userData := payload.UserData
	authID := payload.AuthID

	var violations []skyerr.Cause
	check := func(v *PasswordViolation) {
		if v != nil {
			violations = append(violations, *v)
		}
	}

	check(pc.checkPasswordLength(password))
	check(pc.checkPasswordUppercase(password))
	check(pc.checkPasswordLowercase(password))
	check(pc.checkPasswordDigit(password))
	check(pc.checkPasswordSymbol(password))
	check(pc.checkPasswordExcludedKeywords(password))
	check(pc.checkPasswordExcludedFields(password, userData))
	check(pc.checkPasswordGuessableLevel(password, userData))
	check(pc.checkPasswordHistory(password, authID))

	if len(violations) == 0 {
		return nil
	}

	return PasswordPolicyViolated.NewWithCauses("password policy violated", violations)
}

func (pc *PasswordChecker) ShouldSavePasswordHistory() bool {
	return pc.PasswordHistoryEnabled
}

func (pc *PasswordChecker) shouldCheckPasswordHistory() bool {
	return pc.ShouldSavePasswordHistory()
}

func IsSamePassword(hashedPassword []byte, password string) bool {
	return corepassword.Compare([]byte(password), hashedPassword) == nil
}
