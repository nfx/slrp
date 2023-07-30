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
	"time"

	"github.com/nfx/slrp/pool/counter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	endpoint := flag.String("proxy", "https://localhost:8090", "URL of SRLP installation")
	workers := flag.Int("workers", 2, "number of workers")
	failAfter := flag.Int("failAfter", 1000, "number of workers")
	flag.Parse()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	proxyURL, err := url.Parse(*endpoint)
	if err != nil {
		log.Err(err).Msg("failed to parse Proxy URL")
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
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
			var serial string
			if res != nil && res.Header != nil {
				serial = res.Header.Get("X-Proxy-Serial")
			}
			log.Err(err).Str("serial", serial).Msg("failed to get")
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
		seen := map[string]*counter.RollingCounter{}
		for {
			select {
			case <-ctx.Done():
				return
			case i := <-show:
				if errors > *failAfter {
					log.Error().Int("errors", errors).Msg("stopping")
					cancel()
				}
				if i.status > 430 {
					errors++
					break
				}
				errors = 0
				cnt, ok := seen[i.proxy]
				if !ok {
					*cnt = counter.NewRollingCounter(1, time.Minute)
					seen[i.proxy] = cnt
				}
				cnt.Increment()
				msg := "ok"
				var err error
				rollingSum := cnt.Sum()
				if rollingSum > 1 {
					err = fmt.Errorf("seen %d times", rollingSum)
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
	log.Info().
		Int("workers", *workers).
		Msg("starting")
	<-ctx.Done()
}
