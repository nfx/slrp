package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type envm map[string]string

func (env envm) apply() {
	for k, v := range env {
		os.Setenv(k, v)
	}
}

func (env envm) restore() func() {
	backup := envm{}
	for _, line := range os.Environ() {
		pair := strings.SplitN(line, "=", 2)
		backup[pair[0]] = pair[1]
	}
	os.Clearenv()
	env.apply()
	return func() {
		backup.apply()
	}
}

func TestConfigResolvesFromSlrpYmlInPwd(t *testing.T) {
	testdata, _ := filepath.Abs("testdata")
	defer envm{
		"APP":      "slrp",
		"PWD":      fmt.Sprintf("%s/a", testdata),
		"SLRP_A_B": "1",
	}.restore()()

	config, err := getConfig()
	assert.NoError(t, err)

	a := config["thing"].BoolOr("a", false)
	assert.True(t, a)

	b := config["a"].IntOr("b", 2)
	assert.Equal(t, 1, b)
}

func TestConfigResolvesFromConfigYmlInPwd(t *testing.T) {
	testdata, _ := filepath.Abs("testdata")
	defer envm{
		"APP":      "slrp",
		"PWD":      fmt.Sprintf("%s/b", testdata),
		"SLRP_A_B": "1",
	}.restore()()

	config, err := getConfig()
	assert.NoError(t, err)

	a := config["king"].BoolOr("a", false)
	assert.True(t, a)

	b := config["a"].IntOr("b", 2)
	assert.Equal(t, 1, b)
}

func TestConfigResolvesFromConfigYmlInHomeDotSlrp(t *testing.T) {
	testdata, _ := filepath.Abs("testdata")
	defer envm{
		"APP":      "slrp",
		"HOME":     fmt.Sprintf("%s/c", testdata),
		"SLRP_A_B": "1",
	}.restore()()

	config, err := getConfig()
	assert.NoError(t, err)

	a := config["ping"].BoolOr("a", false)
	assert.True(t, a)

	b := config["a"].IntOr("b", 2)
	assert.Equal(t, 1, b)
}

func TestConfigResolvesFromSlrpYamlError(t *testing.T) {
	testdata, _ := filepath.Abs("testdata")
	defer envm{
		"APP":      "slrp",
		"PWD":     fmt.Sprintf("%s/d", testdata),
		"SLRP_A_B": "1",
	}.restore()()

	_, err := getConfig()
	assert.Error(t, err)
}

func TestConfigEnvBools(t *testing.T) {
	defer envm{
		"ANY_A_B":    "1",
		"SLRP_A_B":   "1",
		"SLRP_A_B_C": "true",
		"SLRP_A_B_D": "10s",
	}.restore()()

	var cfg Config
	def := cfg.BoolOr("new", true)
	assert.True(t, def)

	config, err := getConfig()
	assert.NoError(t, err)

	def = config["a"].BoolOr("new", true)
	assert.True(t, def)

	set := config["a"].BoolOr("b_c", false)
	assert.True(t, set)

	set = config["a"].BoolOr("b_d", false)
	assert.False(t, set)
}

func TestConfigEnvStrs(t *testing.T) {
	defer envm{
		"SLRP_A_B":   "1",
		"SLRP_A_B_C": "true",
		"SLRP_A_B_D": "10s",
	}.restore()()

	var cfg Config
	def := cfg.StrOr("new", "a")
	assert.Equal(t, "a", def)

	config, err := getConfig()
	assert.NoError(t, err)

	def = config["a"].StrOr("new", "a")
	assert.Equal(t, "a", def)

	set := config["a"].StrOr("b", "a")
	assert.Equal(t, "1", set)

	set = config["a"].StrOr("b_d", "a")
	assert.Equal(t, "10s", set)
}

func TestConfigEnvInts(t *testing.T) {
	defer envm{
		"SLRP_A_B":   "1",
		"SLRP_A_B_C": "true",
		"SLRP_A_B_D": "10s",
	}.restore()()

	var cfg Config
	def := cfg.IntOr("new", 10)
	assert.Equal(t, 10, def)

	config, err := getConfig()
	assert.NoError(t, err)

	def = config["a"].IntOr("new", 10)
	assert.Equal(t, 10, def)

	set := config["a"].IntOr("b", 20)
	assert.Equal(t, 1, set)

	set = config["a"].IntOr("b_d", 10)
	assert.Equal(t, 10, set)
}

func TestConfigEnvDurs(t *testing.T) {
	defer envm{
		"SLRP_A_B":   "1",
		"SLRP_A_B_C": "true",
		"SLRP_A_B_D": "10s",
	}.restore()()

	var cfg Config
	def := cfg.DurOr("new", time.Second*20)
	assert.Equal(t, time.Second*20, def)

	config, err := getConfig()
	assert.NoError(t, err)

	def = config["a"].DurOr("new", time.Second*20)
	assert.Equal(t, time.Second*20, def)

	set := config["a"].DurOr("b_d", time.Second*20)
	assert.Equal(t, time.Second*10, set)

	set = config["a"].DurOr("b_c", time.Second*20)
	assert.Equal(t, time.Second*20, set)
}
