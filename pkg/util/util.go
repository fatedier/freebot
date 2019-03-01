package util

func StringContains(strs []string, s string) bool {
	for _, v := range strs {
		if v == s {
			return true
		}
	}
	return false
}
