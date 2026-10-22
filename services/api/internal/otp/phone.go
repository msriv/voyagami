package otp

// ValidPhone reports whether s looks like an E.164 number: a plus sign, then
// 8 to 15 digits with no leading zero. Region rules stay with the provider.
func ValidPhone(s string) bool {
	if len(s) < 9 || len(s) > 16 || s[0] != '+' || s[1] == '0' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
