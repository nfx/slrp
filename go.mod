module github.com/nfx/slrp

go 1.18

// direct
require (
	github.com/Bogdan-D/go-socks4 v1.0.0
	github.com/PuerkitoBio/goquery v1.8.0
	github.com/alecthomas/participle/v2 v2.0.0-beta.4
	github.com/corpix/uarand v0.2.0
	github.com/dop251/goja v0.0.0-20220124171016-cfb079cdc7b4
	github.com/ghodss/yaml v1.0.0
	github.com/gorilla/mux v1.8.0
	github.com/microcosm-cc/bluemonday v1.0.18
	github.com/oschwald/maxminddb-golang v1.9.0
	github.com/rs/zerolog v1.27.0
	github.com/yosssi/gohtml v0.0.0-20201013000340-ee4748c638f4
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

// test
require (
	github.com/maxmind/mmdbwriter v0.0.0-20220606140952-b99976ab4826
	github.com/stretchr/testify v1.7.2
)

// indirect
require (
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.4.1-0.20201116162257-a2a8dda75c91 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.3.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// TODO: fix yaml dependencies
require gopkg.in/yaml.v2 v2.4.0 // indirect

require (
	github.com/BurntSushi/toml v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go4.org/intern v0.0.0-20211027215823-ae77deb06f29 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20211027215541-db492cf91b37 // indirect
	golang.org/x/sys v0.0.0-20220325203850-36772127a21f // indirect
	inet.af/netaddr v0.0.0-20211027220019-c74959edd3b6 // indirect
)
