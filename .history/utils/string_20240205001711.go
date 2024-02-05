package utils

import "strings"

func RemoveCommas(s string) string {
	return strings.ReplaceAll(s, ",", "")
}
