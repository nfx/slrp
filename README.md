SLRP - rotating open proxy multiplexer
---

![slrp logo](ui/public/logo.png)

* Searches for proxies in open sources
* Intelligently stores state on disk across restarts
* Validates via configurable speed thresholds and anonymity
* Multiplexes HTTP/HTTPS MITM to HTTP, HTTPS, SOCKS4, and SOCKS5
* Exposes REST API for refresh stats and pool health
* Records request history in-memory for further UI inspection
* Real-time statistics display about available pool
* Packaged as a single executable binary, that also includes Web UI

> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

# References

* [ProxyBroker](https://github.com/constverum/ProxyBroker) is pretty similar project in nature. Requires couple of Python module dependencies and had the last commit in March 2019. 
* [Scylla](https://github.com/imWildCat/scylla) is pretty similar project in nature. Requires couple of Python module dependencies.