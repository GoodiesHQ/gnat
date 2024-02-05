package utils

import "strings"

func RemoveCommas(s string) string {
	return strings.ReplaceAll(s, ",", "")
}

func TrimEmptyLines(slice []string) []string {
	beg := 0
	for beg < len(slice) && slice[beg] == "" {
		beg++
	}

	end := len(slice)
	for end > beg && slice[end-1] == "" {
		end--
	}

	return slice[beg:end]
}

func SplitLines(s string) []string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")

}

func JoinLines(s []string) string {
	return strings.Join(s, "\n")
}
