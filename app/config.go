package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/rs/zerolog/log"
)

// Prefix is the env var prefix
var Prefix = "slrp"

func init() {
	os.Setenv("APP", Prefix)
}

var envVar = regexp.MustCompile(`\$([A-Z_]+)`)

func expandEnv(in string) string {
	for _, match := range envVar.FindAllStringSubmatch(in, -1) {
		value := os.Getenv(match[1])
		if value == "" {
			continue
		}
		in = strings.ReplaceAll(in, match[0], value)
	}
	return in
}

type Config map[string]string

type configurable interface {
	Configure(Config) error
}

type configuration map[string]Config

func (c Config) StrOr(key, def string) string {
	if c == nil {
		return expandEnv(def)
	}
	v, ok := c[key]
	if !ok {
		v = def
	}
	return expandEnv(v)
}

func (c Config) DurOr(key string, def time.Duration) time.Duration {
	if c == nil {
		return def
	}
	v, ok := c[key]
	if !ok {
		return def
	}
	d, err := ParseDuration(v)
	if err != nil || d == 0 {
		log.Warn().Err(err).Str("key", key).Msg("cannot parse duration")
		return def
	}
	return d
}

func (c Config) IntOr(key string, def int) int {
	if c == nil {
		return def
	}
	v, ok := c[key]
	if !ok {
		return def
	}
	p, err := strconv.Atoi(v)
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("cannot parse int")
		return def
	}
	return p
}

func (c Config) BoolOr(key string, def bool) bool {
	if c == nil {
		return def
	}
	v, ok := c[key]
	if !ok {
		return def
	}
	switch strings.ToLower(v) {
	case "yes", "true":
		return true
	default:
		return false
	}
}

func getConfig() (configuration, error) {
	var raw []byte
	var err error
	locs := []string{
		path.Clean(expandEnv("$PWD/$APP.yml")),
		path.Clean(expandEnv("$PWD/config.yml")),
		path.Clean(expandEnv("$HOME/.$APP/config.yml")),
	}
	validLoc := ""
	for _, loc := range locs {
		raw, err = ioutil.ReadFile(loc)
		if os.IsNotExist(err) {
			continue
		}
		validLoc = loc
		break
	}
	os.Stderr.WriteString(fmt.Sprintf("Loading config: %s\n", validLoc))
	data := configuration{}
	err = yaml.Unmarshal(raw, &data)
	if err != nil {
		return nil, fmt.Errorf("invalid config in %s: %w", validLoc, err)
	}
	if data == nil {
		// when there's no config
		data = configuration{}
	}
	for _, raw := range os.Environ() {
		rawSplit := strings.SplitN(raw, "=", 2)
		v := strings.ToLower(rawSplit[0])
		s := strings.SplitN(v, "_", 3)
		if len(s) != 3 {
			continue
		}
		if s[0] != Prefix {
			continue
		}
		component, key := s[1], s[2]
		c, ok := data[component]
		if ok {
			c[key] = rawSplit[1]
		} else {
			data[component] = Config{key: rawSplit[1]}
		}
	}
	return data, nil
}
