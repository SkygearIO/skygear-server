// Code generated by "stringer -type=PasswordViolationReason"; DO NOT EDIT.

package audit

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[PasswordTooShort-0]
	_ = x[PasswordUppercaseRequired-1]
	_ = x[PasswordLowercaseRequired-2]
	_ = x[PasswordDigitRequired-3]
	_ = x[PasswordSymbolRequired-4]
	_ = x[PasswordContainingExcludedKeywords-5]
	_ = x[PasswordBelowGuessableLevel-6]
	_ = x[PasswordReused-7]
	_ = x[PasswordExpired-8]
}

const _PasswordViolationReason_name = "PasswordTooShortPasswordUppercaseRequiredPasswordLowercaseRequiredPasswordDigitRequiredPasswordSymbolRequiredPasswordContainingExcludedKeywordsPasswordBelowGuessableLevelPasswordReusedPasswordExpired"

var _PasswordViolationReason_index = [...]uint8{0, 16, 41, 66, 87, 109, 143, 170, 184, 199}

func (i PasswordViolationReason) String() string {
	if i < 0 || i >= PasswordViolationReason(len(_PasswordViolationReason_index)-1) {
		return "PasswordViolationReason(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _PasswordViolationReason_name[_PasswordViolationReason_index[i]:_PasswordViolationReason_index[i+1]]
}
