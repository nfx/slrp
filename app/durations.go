package app

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var (
	biggerDurationsRE = regexp.MustCompile(`(?m)(\d+)([wdhms])`)
	conv              = map[string]time.Duration{
		"s": time.Second,
		"m": time.Minute,
		"h": time.Hour,
		"d": 24 * time.Hour,
		"w": 7 * 24 * time.Hour,
	}
)

// maybe not the most performant duration parser, but at least it supports days and weeks
func ParseDuration(dur string) (result time.Duration, err error) {
	for _, match := range biggerDurationsRE.FindAllStringSubmatch(dur, -1) {
		incr, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, err
		}
		multiple, ok := conv[match[2]]
		if !ok {
			return 0, fmt.Errorf("cannot find multiple: %s", match[2])
		}
		result += time.Duration(incr) * multiple
	}
	return
}
