package app

import (
	"context"
	"encoding"
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"

	_ "net/http/pprof"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.DurationFieldUnit = time.Second
}

type Fabric struct {
	State     string
	Factories Factories

	singletons    Singletons
	services      map[string]Service
	contexts      map[string]*serviceContext
	updated       map[string]time.Time
	flushed       map[string]time.Time
	configuration configuration

	syncService chan string
	askStats    chan chan stats
	syncTrigger *time.Ticker
}

type stat struct {
	Endpoint string `json:",omitempty"`
	Updated  time.Time
	Flushed  time.Time
}

type stats map[string]stat

type aServer interface {
	ListenAndServe() error
	Close() error
}

func Run(ctx context.Context, f Factories) {
	(&Fabric{Factories: f}).Start(ctx)
}

func (f *Fabric) Start(ctx context.Context) {
	f.syncService = make(chan string)
	f.askStats = make(chan chan stats)
	f.updated = map[string]time.Time{}
	f.flushed = map[string]time.Time{}
	f.contexts = map[string]*serviceContext{}
	f.services = map[string]Service{}
	f.loadConfiguration()
	f.initLogging()
	// embedded UI needs server router to attach to
	f.Factories["server"] = newServer
	// server needs fabric
	f.Factories["fabric"] = func() *Fabric {
		return f
	}
	// and every dependency would just recursively resolve
	f.singletons = f.Factories.Init()
	syncTrigger := f.configuration["app"].DurOr("sync", 1*time.Minute)
	f.syncTrigger = time.NewTicker(syncTrigger)
	f.State = f.configuration["app"].StrOr("state", "$HOME/.$APP/data")

	if f.configuration["pprof"].BoolOr("enable", false) {
		addr := f.configuration["pprof"].StrOr("addr", "localhost:6060")
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
		f.singletons["pprof"] = &http.Server{Addr: addr}
		log.Info().Str("addr", addr).Msg("Enabled pprof")
	}

	monitor := f.singletons.Monitor()

	// treat all ListenAndServe exposing singletons as another service
	f.services["monitor"] = monitor

	f.initServices()
	f.configureServices()
	f.loadState()
	f.startAll(ctx)
	go f.sync(ctx)

	// wait for all servers to stop
	monitor.Wait()
}

func (f *Fabric) initLogging() {
	levels := map[string]zerolog.Level{
		"trace": zerolog.TraceLevel,
		"debug": zerolog.DebugLevel,
		"info":  zerolog.InfoLevel,
		"warn":  zerolog.WarnLevel,
	}
	logLevel := f.configuration["log"].StrOr("level", "info")
	level, ok := levels[logLevel]
	if !ok {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	logFormat := f.configuration["log"].StrOr("format", "pretty")
	switch logFormat {
	case "pretty":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	case "json":
		log.Logger = log.Output(os.Stdout)
	case "file":
		var lb lumberjack.Logger
		// TODO: make it more configurable
		lb.Filename = f.configuration["log"].StrOr("file", "$PWD/$APP.log")
		lb.MaxBackups = 0
		log.Logger = log.Output(&lb)
	}
}

func (f *Fabric) loadConfiguration() {
	conf, err := getConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load configuration")
	}
	f.configuration = conf
}

type monitorServers struct {
	sync.WaitGroup
	Singletons
}

func (m *monitorServers) Start(ctx Context) {
	for s, v := range m.Singletons {
		srv, ok := v.(aServer)
		if !ok {
			continue
		}
		m.Add(1)
		go m.closeOnDone(ctx.Done(), s, srv)
		go m.listenAndServe(s, srv)
	}
}

func (m *monitorServers) closeOnDone(done <-chan struct{}, service string, srv aServer) {
	<-done
	err := srv.Close()
	log.Warn().Str("service", service).Err(err).Msg("parent context done")
}

func (m *monitorServers) listenAndServe(service string, server aServer) {
	log.Info().Str("service", service).Msg("starting")
	err := server.ListenAndServe()
	log.Warn().Str("service", service).Err(err).Msg("stopped")
	m.Done()
}

func (f *Fabric) startAll(ctx context.Context) {
	for service := range f.services {
		log.Debug().Str("service", service).Msg("starting")
		f.contexts[service] = &serviceContext{
			ctx:  ctx,
			sync: f.syncService,
			name: service,
		}
		f.services[service].Start(f.contexts[service])
	}
	log.Debug().Msg("all services loaded")
}

func (f *Fabric) loadState() {
	for service := range f.services {
		_, ok := f.services[service].(encoding.BinaryUnmarshaler)
		if !ok {
			continue
		}
		err := f.load(service)
		if os.IsNotExist(err) {
			continue
		}
		log.Err(err).Str("service", service).Msg("loaded")
	}
}

func (f *Fabric) configureServices() {
	for service, s := range f.singletons {
		c, ok := s.(configurable)
		if !ok {
			continue
		}
		err := c.Configure(f.configuration[service])
		if err != nil {
			log.Fatal().Err(err).
				Str("service", service).
				Msg("cannot configure")
		}
	}
}

func (f *Fabric) initServices() {
	for k, v := range f.singletons {
		srv, ok := v.(Service)
		if !ok {
			continue
		}
		f.services[k] = srv
	}
}

func (h *Fabric) snapshot() stats {
	resp := make(chan stats)
	h.askStats <- resp
	res := <-resp
	return res
}

func (h *Fabric) sync(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case resp := <-h.askStats:
			s := stats{}
			for k := range h.services {
				s[k] = stat{
					Updated: h.updated[k],
					Flushed: h.flushed[k],
				}
			}
			resp <- s
		case service := <-h.syncService:
			// keep service up-to-date time
			h.updated[service] = time.Now()
		case <-h.syncTrigger.C:
			for service, updated := range h.updated {
				// some services don't need to have a persistent state
				_, ok := h.services[service].(encoding.BinaryMarshaler)
				if !ok {
					continue
				}
				flushed := h.flushed[service]
				if flushed.After(updated) {
					continue
				}
				// flush state to disk only for those changed
				log.Info().Str("service", service).Msg("flushing")
				// theoretically we can make it transactional, but it'll
				// block all heartbeat producers, so don't really need it now
				go h.flush(service)
				// add a channel to ensure that flush succeeded
				h.flushed[service] = time.Now()
			}
		}
	}
}

func (h *Fabric) load(service string) error {
	l := func(file string) error {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		dec := gob.NewDecoder(f)
		// TODO: check loads!!
		return dec.Decode(h.services[service])
	}
	db := fmt.Sprintf("%s/%s", h.State, service)
	err := l(db)
	if err != nil {
		err = l(db + ".bak")
	}
	return err
}

func (h *Fabric) flush(service string) {
	err := os.MkdirAll(h.State, 0700)
	if err != nil {
		log.Err(err).Msg("cannot create folder")
		return
	}
	db := fmt.Sprintf("%s/%s", h.State, service)
	if _, err := os.Stat(db); err == nil {
		err = os.Rename(db, db+".bak")
		if err != nil {
			log.Warn().Err(err).Str("service", service).Msg("cannot sync backup")
		}
	}
	f, err := os.Create(db)
	if err != nil {
		log.Warn().Err(err).Str("service", service).Msg("cannot sync")
		return
	}
	gob.NewEncoder(f).Encode(h.services[service])
	if err != nil {
		log.Warn().Err(err).Str("service", service).Msg("cannot sync backup")
	}
	f.Close()
	log.Info().Str("file", db).Str("service", service).Msg("synced state")
}
