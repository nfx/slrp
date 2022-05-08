module github.com/nfx/slrp

go 1.18

// direct
require (
	github.com/Bogdan-D/go-socks4 v1.0.0
	github.com/PuerkitoBio/goquery v1.7.1
	github.com/alecthomas/participle/v2 v2.0.0-alpha7
	github.com/corpix/uarand v0.1.1
	github.com/dop251/goja v0.0.0-20220124171016-cfb079cdc7b4
	github.com/ghodss/yaml v1.0.0
	github.com/gorilla/mux v1.8.0
	github.com/microcosm-cc/bluemonday v1.0.18
	github.com/rs/zerolog v1.26.1
	github.com/yosssi/gohtml v0.0.0-20201013000340-ee4748c638f4
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

// test
require github.com/stretchr/testify v1.7.1

// indirect
require (
	github.com/andybalholm/cascadia v1.2.0 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.4.1-0.20201116162257-a2a8dda75c91 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.3.6 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

// TODO: fix yaml dependencies
require gopkg.in/yaml.v2 v2.4.0 // indirect

require github.com/BurntSushi/toml v1.0.0 // indirect
