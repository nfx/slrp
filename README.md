SLRP - rotating open proxy multiplexer
---

![slrp logo](ui/public/logo.png)

![lines](https://img.shields.io/tokei/lines/github/nfx/slrp)
[![downloads](https://img.shields.io/github/downloads/nfx/slrp/total.svg)](https://hanadigital.github.io/grev/?user=nfx&repo=slrp)

* Searches for proxies in open sources
* Intelligently stores state on disk across restarts
* Validates via configurable speed thresholds and anonymity
* Multiplexes HTTP/HTTPS MITM to HTTP, HTTPS, SOCKS4, and SOCKS5
* Exposes REST API for refresh stats and pool health
* Exposes minimal Query Language for filtering of History and Proxy Stats.
* Records request history in-memory for further UI inspection
* Real-time statistics display about available pool
* Packaged as a single executable binary, that also includes Web UI

# Concepts

* *Source* is an async process that looks at one or more pages for refreshed proxy list. 
* *Refresher* component does best effort on *scheduling* items. 
* Some *sources* perform better forwarded through a *Pool*, warming it up.
* One *proxy* may be seen in multiple *sources*, so we keep *exclusive* proxies per source across refreshes, which are not found in other sources.
* *Proxy* consists of protocol (HTTP, HTTPS, SOCKS4, or SOCKS5) and IP:PORT.
* *Proxy* becomes *Scheduled* immediately after it's seen in the source.
* *Scheduled* could transition into *Probing* queue if it's not *Ignored* (e.g. *Timeouts* or *Blacklist*).
* *Probing* uses configurable pool of rotating anonymity *checkers* to check for liveliness.
* *Timeout* items are re-added to *Scheduled* queue as *Reverify* source to probe item up to 5 times.
* *Blacklist* hosts historical faulty proxies that should never be probed again.
* Successful *check* results in *Found* queue and gets added to a *Pool*.
* *Pool* subdivides its memory into *shards* for randomized rotation and minimal resource contention.
* *Pool* uses configurable and backpressure-controlled workers to perform HTTP request forwarding.
* Every *forwarded request* gets a serial number and picks a different *shard* for an *attempt*.
* Every *forwarded request* can later be inspected through `GET /api/history` or UI.
* Every *attempt* picks first available working random proxy from a *shard* and marks it as *Offered*.
* In the event of no working proxies in a *shard*, *proxy pool exhaustion* errors can do backpressure and slow down issuing of *serial* numbers through simple leaky bucket algorithm.
* Every *succeeded attempt* through a proxy increases it's *Success Rate* (*Succeeded*/*Offered*), which is also calculated per hour.
* Every *failed attempt* marks proxy as not working and *suspends offering* it for 5 minutes.

> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

# User Interface

## Overview

![overview](docs/overview.png)

Shows current source refresh status and stats.

## Proxies

![proxies](docs/proxies.png)

Search interface over active pool of found proxies. By default, entries are sorted by last working on top.

## History

![history](docs/history.png)

Search interface over last 1000 forwarding attempts (configurable).

# Configuration

Conf file is looked in the following paths:

1. `$PWD/slrp.yml`
2. `$PWD/config.yml`
3. `$HOME/.slrp/config.yml`

Default configuration is approximately the following:

```yaml
app:
  state: $HOME/.slrp/data
  sync: 1m
log:
  level: info
  format: pretty
server:
  addr: "127.0.0.1:8089"
  read_timeout: 15s
  enable_profiler: false
checker:
  timeout: 5s
  strategy: simple
history:
  limit: 1000
```

Every configuration property can be overridden through environment variable by using `SLRP_` prefix followed by section name and key, divided by `_`. For example, in order to set log level to trace, do `SLRP_LOG_LEVEL=TRACE slrp`.

## app

Fabric that holds application components together.

* `state` - where data persists on disk through restarts of the application. Default is `.slrp/data` of your home directory.
* `sync` - how often data is synchronised to disk, pending availability of any updates of component state. Default is every minute.

## log

Structured logging meta-components.

* `level` - log level of application. Default is `info`. Possible values are `trace`, `debug`, `info`, `warn`, and `error`.
* `format` - format of log lines printed. Default is `pretty`, though it's recommended for exploratory use only for performance reasons. Possible values are `pretty`, `json`, and `file` _(experimental)_. `file` will create a `$PWD/slrp.log`, unless specified by `log.file` property.
* `file` _(experimental)_ - application logs in JSON format. Default value is `$PWD/slrp.log`.

## server

API and UI serving component.

* `addr` - address of listening HTTP server. Default is [http://127.0.0.1:8089](http://127.0.0.1:8089).
* `read_timeout` - default is `15s`.
* `enable_profiler` - either or not enabling profiler endpoints. Default is `false`. Developer use only.

## checker

Component for verification of proxy liveliness and anonymity.

* `timeout` - time to wait while performing verificatin. Default is `5s`.
* `strategy` - verification strategy to check the IP of the proxy. Default is `simple`, which will randomly select one of publicly available sites: [ifconfig.me](https://ifconfig.me), [ifconfig.io](https://ifconfig.io), [myexternalip.com](https://myexternalip.com), [ipv4.icanhazip.com/](https://ipv4.icanhazip.com/), [https://ipinfo.io/](ipinfo.io/), [api.ipify.org/](https://api.ipify.org/), or [wtfismyip.com](https://wtfismyip.com). Another strategy is `headers`, which will look for the real IP address in [https://ifconfig.me/all](https://ifconfig.me/all) or [https://ifconfig.io/all.json](https://ifconfig.io/all.json), which might have been added in HTTP headers while forwarding. And there's `twopass` strategy, that will first perform `simple` check and `headers` afterwards.

## history

Component for recording forwarded requests through a pool of proxies.

* `limit` - number of requests to keep in memory. Default is `1000`.

# API

## GET `/api`

Retrieve last sync status for all components

## GET `/api/dashboard`

Get information about refresh status for all sources

## GET `/api/pool`

Get 20 last used proxies

## GET `/api/history`

Get 100 last forwarding attempts

## GET `/api/history/:id`

Get sanitized HTTP response from forwarding attempt

## GET `/api/blacklist`

Get first 20 blacklisted items sorted by proxy along with common error stats

# References

* [ProxyBroker](https://github.com/constverum/ProxyBroker) is pretty similar project in nature. Requires couple of Python module dependencies and had the last commit in March 2019. 
* [Scylla](https://github.com/imWildCat/scylla) is pretty similar project in nature. Requires couple of Python module dependencies.
