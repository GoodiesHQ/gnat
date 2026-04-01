package driver

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	reUptimeYears = regexp.MustCompile(`(\d+)\s+years?`)
	reUptimeWeeks = regexp.MustCompile(`(\d+)\s+weeks?`)
	reUptimeDays  = regexp.MustCompile(`(\d+)\s+days?`)
	reUptimeHours = regexp.MustCompile(`(\d+)\s+h(?:rs?|ours?)`)
	reUptimeMins  = regexp.MustCompile(`(\d+)\s+min`)
)

// parseUptimeSeconds converts a free-form uptime string into total seconds.
func ParseUptimeSeconds(s string) (int64, error) {
	units := []struct {
		re     *regexp.Regexp
		factor int64
	}{
		{reUptimeYears, 365 * 24 * 3600},
		{reUptimeWeeks, 7 * 24 * 3600},
		{reUptimeDays, 24 * 3600},
		{reUptimeHours, 3600},
		{reUptimeMins, 60},
	}
	var total int64
	matched := false
	for _, u := range units {
		if m := u.re.FindStringSubmatch(s); len(m) > 1 {
			n, _ := strconv.ParseInt(m[1], 10, 64)
			total += n * u.factor
			matched = true
		}
	}
	if !matched {
		return 0, fmt.Errorf("no recognized time units in uptime string: %q", s)
	}
	return total, nil
}
