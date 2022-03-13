package stats

import (
	"bytes"
	"encoding/gob"
	"net/http"
	"time"

	"github.com/nfx/slrp/app"
)

type state string

const (
	Idle    state = "idle"
	Running state = "running"
	Failed  state = "failed"
)

type Stat struct {
	Name        string `json:",omitempty"` // TODO: this is a hack?...
	State       state
	Progress    int
	anticipated int

	Scheduled   int
	New         int
	Probing     int
	Found       int
	Timeouts    int
	Blacklisted int
	Ignored     int
	Updated     time.Time
	finished    bool

	Failure string
}

func (v Stat) Pipeline() int {
	return v.Scheduled + v.New + v.Probing
}

func (v Stat) Processed() int {
	return v.Found + v.Timeouts + v.Blacklisted + v.Ignored
}

type increment int

const (
	Launched increment = iota
	Scheduled
	Ignored
	Probing
	New
	Timeout
	Blacklisted
	Found
	Finished
)

// update is a "fat" model to reduce number of channels
type update struct {
	sourceId    int
	state       increment
	err         error
	anticipated int
}

type Stats struct {
	sources  map[int]*Stat
	snapshot chan chan Sources
	updates  chan update
}

func NewStats() *Stats {
	return &Stats{
		updates:  make(chan update),
		sources:  make(map[int]*Stat),
		snapshot: make(chan chan Sources),
	}
}

func (s *Stats) Start(ctx app.Context) {
	go s.main(ctx)
}

func (s *Stats) Launch(source int) {
	s.updates <- update{
		sourceId: source,
		state:    Launched,
	}
}

func (s *Stats) LaunchAnticipated(source, count int) {
	s.updates <- update{
		sourceId:    source,
		state:       Launched,
		anticipated: count,
	}
}

func (s *Stats) Update(source int, state increment) {
	s.updates <- update{
		sourceId: source,
		state:    state,
	}
}

func (s *Stats) Finish(source int, err error) {
	s.updates <- update{
		sourceId: source,
		state:    Finished,
		err:      err,
	}
}

func (s *Stats) Snapshot() Sources {
	req := make(chan Sources)
	defer close(req)
	s.snapshot <- req
	return <-req
}

func (s *Stats) HttpGet(_ *http.Request) (interface{}, error) {
	return s.Snapshot(), nil
}

func (s *Stats) handleUpdate(u update) {
	stat, ok := s.sources[u.sourceId]
	if !ok {
		s.sources[u.sourceId] = &Stat{
			State: Running,
		}
		stat = s.sources[u.sourceId]
	}
	switch u.state {
	case Launched:
		if u.anticipated == 0 {
			u.anticipated = stat.Processed()
		}
		stat = &Stat{
			// heuristics based on previous run
			anticipated: u.anticipated,
			State:       Running,
		}
		s.sources[u.sourceId] = stat
	case Scheduled:
		stat.Scheduled++
	case Ignored:
		stat.Scheduled--
		stat.Ignored++
	case New:
		stat.Scheduled--
		stat.New++
	case Probing:
		stat.New--
		stat.Probing++
	case Found:
		stat.Probing--
		stat.Found++
	case Timeout:
		stat.Probing--
		stat.Timeouts++
	case Blacklisted:
		stat.Probing--
		stat.Blacklisted++
	case Finished:
		stat.finished = true
		if u.err != nil {
			stat.State = Failed
			stat.Failure = u.err.Error()
		}
	}
	if stat.anticipated > 0 {
		processed := stat.Processed()
		total := stat.anticipated
		stat.Progress = int(100 * float32(processed) / float32(total))
	}
	if stat.Progress > 100 {
		// we don't want values like 343%
		stat.Progress = 100
	}
	if stat.finished && stat.State != Failed && stat.Pipeline() == 0 {
		stat.Progress = 100
		stat.State = Idle
	}
	stat.Updated = time.Now()
}

func (s *Stats) main(ctx app.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case u := <-s.updates:
			s.handleUpdate(u)
			ctx.Heartbeat()
		case res := <-s.snapshot:
			snapshot := Sources{}
			for k, v := range s.sources {
				snapshot[k] = *v
			}
			res <- snapshot
		}
	}
}

func (s *Stats) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	snapshot := s.Snapshot()
	gob.NewEncoder(&b).Encode(snapshot)
	return b.Bytes(), nil
}

func (s *Stats) UnmarshalBinary(data []byte) error {
	b := bytes.NewReader(data)
	err := gob.NewDecoder(b).Decode(&s.sources)
	if err != nil {
		return err
	}
	// cleanup that was in progress and didn't finish
	for k, v := range s.sources {
		if v.State != Idle {
			// o_O WTF and no concurrent modification error?..
			delete(s.sources, k)
		}
	}
	return nil
}
