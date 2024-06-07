package utils

func Contains(s string, list []string) bool {
	for _, c := range list {
		if c == s {
			return true
		}
	}
	return false
}
