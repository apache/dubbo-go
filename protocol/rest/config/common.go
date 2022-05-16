package config

// function will return first not empty string ..
func getNotEmptyStr(args ...string) string {
	var r string
	for _, t := range args {
		if len(t) > 0 {
			r = t
			break
		}
	}
	return r
}
