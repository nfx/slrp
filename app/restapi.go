package app

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type httpGet interface {
	HttpGet(*http.Request) (any, error)
}

type httpGetByID interface {
	HttpGetByID(string, *http.Request) (any, error)
}

type httpDeleteByID interface {
	HttpDeletetByID(string, *http.Request) (any, error)
}

type errorBody struct {
	Message string
}

type httpResource struct {
	service    string
	get        httpGet
	getByID    httpGetByID
	deleteByID httpDeleteByID
}

type InternalError struct{ error }

func (i InternalError) Unwrap() error {
	return i.error
}

type NotFound string

func (nf NotFound) Error() string {
	return string(nf)
}

func (hr *httpResource) err(rw http.ResponseWriter, err error) {
	switch err.(type) {
	case InternalError:
		rw.WriteHeader(500)
	case NotFound:
		rw.WriteHeader(404)
	default:
		rw.WriteHeader(400)
	}
	errBody, _ := json.Marshal(errorBody{err.Error()})
	rw.Write(errBody)
}

func (hr *httpResource) recover(rw http.ResponseWriter) {
	p := recover()
	if err, ok := p.(error); ok {
		log.Err(err).Str("service", hr.service).Msg("panic")
		hr.err(rw, InternalError{err})
		return
	}
	if p != nil {
		hr.err(rw, InternalError{fmt.Errorf("very wrong error")})
		log.Error().
			Interface("panic", p).
			Str("service", hr.service).
			Msg("very wrong panic!")
		return
	}
}

func (hr *httpResource) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	defer hr.recover(rw)
	var response any
	var err error
	if hr.get != nil {
		response, err = hr.get.HttpGet(r)
	} else if hr.getByID != nil {
		vars := mux.Vars(r) // id is defined by the route
		response, err = hr.getByID.HttpGetByID(vars["id"], r)
	} else if hr.deleteByID != nil {
		vars := mux.Vars(r) // id is defined by the route
		response, err = hr.deleteByID.HttpDeletetByID(vars["id"], r)
	}
	if err != nil {
		hr.err(rw, err)
		return
	}
	if response == nil {
		rw.WriteHeader(200)
		return
	}
	if r.FormValue("format") == "text" {
		rw.WriteHeader(200)
		rw.Write([]byte(fmt.Sprintf("%s", response)))
		return
	}
	body, err := json.Marshal(response)
	if err != nil {
		hr.err(rw, err)
		return
	}
	rw.WriteHeader(200)
	rw.Write(body)
}

type mainServer struct {
	http.Server
	listener net.Listener
	fabric   *Fabric
	router   *mux.Router
	onInit   []func(router *mux.Router)
}

func newServer(fabric *Fabric) *mainServer {
	router := mux.NewRouter()
	return &mainServer{
		fabric: fabric,
		router: router,
		Server: http.Server{
			Handler: router,
		},
	}
}

func (s *mainServer) Configure(c Config) (err error) {
	s.Addr = c.StrOr("addr", "localhost:8089")
	timeout := c.DurOr("read_timeout", 15*time.Second)
	s.ReadTimeout = timeout
	s.IdleTimeout = timeout
	s.WriteTimeout = timeout
	s.listener, err = net.Listen("tcp", s.Addr)
	return err
}

func (s *mainServer) url() string {
	return fmt.Sprintf("http://%s", s.listener.Addr())
}

func (s *mainServer) ListenAndServe() error {
	// this method is required for unit testing mode to be able to
	// listen on a random localhost port during assertiong
	log.Info().Str("endpoint", s.url()).Msg("started REST API")
	return s.Serve(s.listener)
}

func (s *mainServer) Start(ctx Context) {
	// it's easier to lazily init serve mux,
	// rather than tinker with DI container
	s.initRestAPI()
}

func (s *mainServer) initRestAPI() {
	hasApi := map[string]bool{}
	for service, v := range s.fabric.singletons {
		get, ok := v.(httpGet)
		if ok {
			hasApi[service] = true
			s.router.Handle(fmt.Sprintf("/api/%s", service), &httpResource{
				service: service,
				get:     get,
			}).Methods("GET")
		}
		getByID, ok := v.(httpGetByID)
		if ok {
			hasApi[service] = true
			s.router.Handle(fmt.Sprintf("/api/%s/{id}", service), &httpResource{
				service: service,
				getByID: getByID,
			}).Methods("GET")
		}
		deleteByID, ok := v.(httpDeleteByID)
		if ok {
			hasApi[service] = true
			s.router.Handle(fmt.Sprintf("/api/%s/{id}", service), &httpResource{
				service:    service,
				deleteByID: deleteByID,
			}).Methods("DELETE")
		}
	}
	s.router.HandleFunc("/api", func(rw http.ResponseWriter, r *http.Request) {
		snapshot := s.fabric.snapshot()
		for k, v := range snapshot {
			if !hasApi[k] {
				continue
			}
			v.Endpoint = fmt.Sprintf("http://%s/api/%s", r.Host, k)
			snapshot[k] = v
		}
		body, _ := json.MarshalIndent(snapshot, "", "  ")
		rw.WriteHeader(200)
		rw.Write(body)
	})
	for _, cb := range s.onInit {
		cb(s.router)
	}
}

func (s *mainServer) OnInit(cb func(router *mux.Router)) {
	s.onInit = append(s.onInit, cb)
}
