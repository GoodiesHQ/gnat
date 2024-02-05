package utils

import "strings"

func RemoveCommas(s string) string {
	return strings.ReplaceAll(s, ",", "")
}

func SplitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

func JoinLines(s []string) string {
	return strings.Join(s, "\n")
}
