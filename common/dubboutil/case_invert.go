package dubboutil

import "unicode"

func SwapCaseFirstRune(s string) string {
	if s == "" {
		return s
	}

	// convert str to rune slice
	r := []rune(s)
	if unicode.IsUpper(r[0]) {
		r[0] = unicode.ToLower(r[0])
	} else {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}
