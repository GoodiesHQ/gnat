package utils

import (
	"bytes"
	"regexp"
	"strings"
)

// ansiSequences matches ANSI/VT100 terminal escape sequences.
var ansiSequences = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

// NormalizeLineEndings converts \r\n and bare \r to \n so prompt regexes work
func NormalizeLineEndings(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	b = bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
	return b
}

func StripANSI(b []byte) []byte {
	return ansiSequences.ReplaceAll(b, nil)
}

// sanitize strips ANSI codes, normalizes line endings, removes the trailing prompt
// line, and trims whitespace from raw device output.
//
// The entire last line is dropped (back to the preceding newline) rather than just
// the regex match, because HP-style ANSI cursor sequences leave the echoed command
// concatenated with the prompt on one line (e.g. "no pageDLN-Core# ").
func Sanitize(raw []byte, prompts []*regexp.Regexp) string {
	s := string(NormalizeLineEndings(StripANSI(raw)))

	for _, p := range prompts {
		all := p.FindAllStringIndex(s, -1)
		if len(all) == 0 {
			continue
		}
		matchStart := all[len(all)-1][0]
		if nl := strings.LastIndexByte(s[:matchStart], '\n'); nl >= 0 {
			s = s[:nl]
		} else {
			s = "" // prompt is on the only line — no content to preserve
		}
		break
	}

	return strings.TrimSpace(s)
}
