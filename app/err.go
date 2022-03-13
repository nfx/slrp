package app

import (
	"regexp"
	"strings"
)

type stripRule struct {
	r   *regexp.Regexp
	sub string
}

func rule(r, sub string) stripRule {
	return stripRule{regexp.MustCompile(r), sub}
}

var strip = []stripRule{
	rule(`\\`, " "),
	rule(`&[^;]+;`, " "),
	rule(`(?m)\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{2,5}`, "addr:port"),
	rule(`(?m)\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "ip"),
	rule(`(?m)https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&\/\/=]*)`, "url"),
	rule(`"`, ""),
	rule(`[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}`, "host"),
	rule(`:[0-9]{2,5}`, ":port"),
	rule(`addr:port->host:port`, "conn"),
	rule(`addr:port->addr:port`, "conn"),
	rule(` :`, ":"),
	rule(`\s+`, " "),
	rule(`Get url: `, ""),
}

func Shrink(body string) string {
	for _, r := range strip {
		body = r.r.ReplaceAllString(body, r.sub)
	}
	return strings.Trim(body, " ")
}

type Err struct {
	shrink string
	Err    error
}

func (err Err) Error() string {
	return err.shrink
}

func ShErr(err error) Err {
	return Err{
		shrink: Shrink(err.Error()),
		Err:    err,
	}
}
