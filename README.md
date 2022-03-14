SLRP - rotating open proxy multiplexer
---

![slrp logo](ui/public/logo.png)

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

# References

* [ProxyBroker](https://github.com/constverum/ProxyBroker) is pretty similar project in nature. Requires couple of Python module dependencies and had the last commit in March 2019. 
* [Scylla](https://github.com/imWildCat/scylla) is pretty similar project in nature. Requires couple of Python module dependencies.