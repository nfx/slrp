package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	workers := flag.Int("workers", 2, "number of workers")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				Host:   "localhost:8090",
			}),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	type info struct {
		proxy, attempt, offered, succeed, serial string
		status                                   int
	}
	ping := func() info {
		checkers := []string{
			"https://ifconfig.me/ip",
			"https://ifconfig.io/ip",
			"https://myexternalip.com/raw",
			"https://ipv4.icanhazip.com/",
			"https://ipinfo.io/ip",
			"https://api.ipify.org/",
			"https://wtfismyip.com/text",
		}
		choice := rand.Intn(len(checkers))
		res, err := client.Get(checkers[choice])
		if err != nil {
			log.Err(err).Msg("failed to get")
			return info{status: 500}
		}
		defer res.Body.Close()
		ioutil.ReadAll(res.Body)
		return info{
			proxy:   res.Header.Get("X-Proxy-Through"),
			attempt: res.Header.Get("X-Proxy-Attempt"),
			offered: res.Header.Get("X-Proxy-Offered"),
			succeed: res.Header.Get("X-Proxy-Succeed"),
			serial:  res.Header.Get("X-Proxy-Serial"),
			status:  res.StatusCode,
		}
	}
	show := make(chan info, *workers*2)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(*workers)
	for i := 0; i < *workers; i++ {
		go func() {
			for {
				show <- ping()
			}
		}()
	}
	go func() {
		var errors int
		seen := map[string]int{}
		for {
			select {
			case <-ctx.Done():
				return
			case i := <-show:
				if errors > 100 {
					cancel()
				}
				if i.status > 430 {
					errors++
					break
				}
				errors = 0
				seen[i.proxy] = seen[i.proxy] + 1
				msg := "ok"
				var err error
				if seen[i.proxy] > 1 {
					err = fmt.Errorf("seen %d times", seen[i.proxy])
					msg = "not ok"
				}
				log.Info().
					Err(err).
					Str("proxy", i.proxy).
					Str("attempt", i.attempt).
					Str("offered", i.offered).
					Str("succeed", i.succeed).
					Str("serial", i.serial).
					Msg(msg)
			}
		}
	}()
	<-ctx.Done()
}
