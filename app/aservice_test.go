package app

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

func newServiceA() *serviceA {
	return &serviceA{
		state:   make(chan []byte),
		loaded:  make(chan error),
		flushed: make(chan error),
		Number:  100500,
	}
}

type serviceA struct {
	state   chan []byte
	loaded  chan error
	flushed chan error
	Number  int
}

func (a *serviceA) Configure(c Config) (err error) {
	return nil
}

func (a *serviceA) Start(ctx Context) {
	// immediately update a state
	go ctx.Heartbeat()
}

func (a *serviceA) UnmarshalBinary(raw []byte) error {
	log.Info().Msg("waiting to assert state A")
	a.state <- raw
	log.Info().Msg("waiting to load A")
	return <-a.loaded
}

func (a *serviceA) MarshalBinary() ([]byte, error) {
	log.Info().Msg("waiting to flush A")
	return []byte{1}, <-a.flushed
}

func (a *serviceA) HttpGet(*http.Request) (interface{}, error) {
	return 1, nil
}

func (a *serviceA) HttpGetByID(id string, r *http.Request) (interface{}, error) {
	return id, nil
}

func (a *serviceA) HttpDeletetByID(id string, r *http.Request) (interface{}, error) {
	switch id {
	case "error":
		return nil, fmt.Errorf("just error: %s", id)
	case "not-found":
		return nil, NotFound("no ID found")
	case "soft":
		panic(InternalError{fmt.Errorf("panic with error: %s", id)})
	default:
		panic("panic with string")
	}
}
