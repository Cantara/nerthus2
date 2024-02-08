package repeatable

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/consensus"
	"github.com/cantara/gober/stream"
	"github.com/cantara/gober/stream/consumer/competing"
	"github.com/cantara/gober/stream/event"
	"github.com/cantara/gober/stream/event/store"
	"github.com/cantara/gober/sync"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/reader"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/story"
	"github.com/gofrs/uuid"
)

type Reader interface {
	//New([]byte, uuid.UUID) error
	New(string, []byte) error
	Read()
	State() map[string]map[string]State
}

type rr[T any] struct {
	c    competing.Consumer[data]
	s    story.Story
	a    []adapter.Adapter
	cons consensus.Consensus

	dimensions   sync.Map[dimension]
	startAdapter adapter.Adapter
}

type dimension struct {
	state       sync.Obj[State]
	run         sync.Obj[uuid.UUID]
	storyStates sync.Map[storyPart]
}

type storyState struct {
	Start storyPart
	State State
}

type storyPart struct {
	Name  string
	State State
	Next  []*storyPart
	Data  data
}

type State string

const (
	Failed     State = "failed"
	Waiting    State = "waiting"
	Started    State = "started"
	Finished   State = "finished"
	Restarted  State = "restarted"
	Terminated State = "terminated"
	//NotStarted State = "not_started"
)

type data struct {
	Story string
	//Id    uuid.UUID
	Dimension string
	State     State
	Run       uuid.UUID
	Part      string
	Data      []adapter.Data
}

func New[T any](strm stream.Stream, consBuilder consensus.ConsBuilderFunc, ckp stream.CryptoKeyProvider, timeout time.Duration, s story.Story, ctx context.Context, adapters ...adapter.Adapter) (Reader, error) {
	if getAdapter(adapter.Start, adapters) == nil {
		adapters = append(adapters, reader.StartAdapter)
	}
	sa := getAdapter(adapter.Start, adapters)
	if getAdapter(adapter.End, adapters) == nil {
		adapters = append(adapters, reader.EndAdapter)
	}
	m := map[string]struct{}{}
	for _, p := range s.Parts() {
		a := getAdapter(p.Adapter(), adapters)
		if a == nil {
			log.Info("missing adapter", "want", p.Adapter(), "has", adapters)
			return nil, ErrMissingAdapter
		}
		if p.Id() != story.IdEnd {
		reqPrev:
			for _, reqId := range p.Prev() {
				if reqId == story.IdStart {
					continue
				}
				req := getPart(reqId, s.Parts())
				for _, areq := range a.Reqs() {
					if req.Adapter() == areq {
						continue reqPrev
					}
				}
				return nil, fmt.Errorf("story part requires adapter that provided adapter does not require, id: %s, reqid: %s, req: %s, areqs: %v", p.Id(), reqId, req.Adapter(), a.Reqs())
			}
		}
		if p.Id() != story.IdStart {
		reqAdapter:
			for _, areq := range a.Reqs() {
				if areq == "any" {
					continue
				}
				for _, reqId := range p.Prev() {
					req := getPart(reqId, s.Parts())
					if req.Adapter() == areq {
						continue reqAdapter
					}
				}
				return nil, fmt.Errorf("adapter requires adapter that provided story part does not require, id: %s, req: %s", p.Id(), areq)
			}
		}
		m[p.Adapter()] = struct{}{}
	}
	if len(adapters) > len(m) {
		var unused []string
		i := 0
		for _, a := range adapters {
			if _, ok := m[a.Name()]; ok {
				i++
				log.Info(a.Name())
				continue
			}
			unused = append(unused, a.Name())
		}
		return nil, fmt.Errorf("adapters:%v,  %d!=%d!=%d, error:%w", unused, len(adapters), len(m), i, ErrUnusedAdapter)
	}

	datatype := "fairytale_" + strings.ReplaceAll(s.Name(), " ", "-")
	cons, err := consBuilder("reader_"+datatype, timeout)
	if err != nil {
		return nil, err
	}

	c, err := competing.New[data](strm, consBuilder, ckp, store.STREAM_START, datatype, func(v data) time.Duration {
		if v.Story == "" {
			return timeout
		}
		return timeout
	}, ctx)
	if err != nil {
		return nil, err
	}

	r := rr[T]{
		c:            c,
		s:            s,
		a:            adapters,
		cons:         cons,
		startAdapter: sa,
		dimensions:   sync.NewMap[dimension](),
		//run:         sync.NewObj[uuid.UUID](),
		//state:       sync.NewObj[State](),
		//storyStates: sync.NewMap[storyPart](),
	}

	go func() { //Read completed chan and keep a record of all completes and create next events that can now be created
		//comp := sync.NewMap[sync.Map[data]]() //TODO: To create the restartable story, i think i want to reoplace this.
		for e := range c.Completed() {
			d, ok := r.dimensions.Get(e.Data.Dimension)
			if !ok {
				log.Error("missing dimention", "story", s.Name(), "dimension", e.Data.Dimension)
				continue
			}
			if !bytes.Equal(d.run.Get().Bytes(), e.Data.Run.Bytes()) {
				continue
			}
			//r, _ := comp.GetOrInit(e.Data.ReadID.String(), sync.NewMap[data])
			p := s.Part(e.Data.Part)
			d.storyStates.Set(p.Id(), storyPart{
				//Name  string
				State: Finished,
				//Next  []*storyPart
				Data: e.Data,
			})
			r.dimensions.CompareAndSwap(e.Data.Dimension, d, func(stored dimension) bool {
				return bytes.Equal(d.run.Get().Bytes(), stored.run.Get().Bytes())
			})
			//r.Set(p.Id(), e.Data)
		NextPart:
			for _, n := range p.Next() {
				np := s.Part(n)
				td := make([]adapter.Data, len(np.Prev()))
				for i, npr := range np.Prev() {
					state, ok := d.storyStates.Get(npr) //Get(npr)
					if !ok || state.State != Finished || !bytes.Equal(state.Data.Run.Bytes(), e.Data.Run.Bytes()) {
						log.Info("required part not completed", "part", npr)
						continue NextPart
					}
					td[i] = state.Data.Data[0]
				}
				we := event.NewWriteEvent[data](event.Event[data]{
					Type: event.Created,
					Data: data{
						Story:     s.Name(),
						Dimension: e.Data.Dimension,
						Part:      n,
						Run:       e.Data.Run,
						//ReadID: e.Data.ReadID, //To me it seems like not having this would create issues, however, from my test it does not. Might need a longer story
						Data: td, //buf.Bytes(),   //This should become a array of all datas that is required
					},
				})
				consId := fmt.Sprintf("%s_%s", np.Id(), e.Data.Run.String())
				if !cons.Request(consId) {
					continue //Should add to a timeout or something. Could lose events this way
				}
				c.Write() <- we
				status := <-we.Done()
				log.WithError(status.Error).Trace("wrote next part", "story", s.Name(), "part", p.Id(), "next", n)
				if status.Error != nil {
					continue
				}
			}
		}
	}()

	return &r, nil
}

func (r *rr[T]) New(dim string, b []byte) error { //, id uuid.UUID) error {
	run := uuid.Must(uuid.NewV7()) //TODO: This needs to be written to a stream and preferably cancel all competing events.
	d, _ := r.dimensions.GetOrInit(dim, func() dimension {
		return dimension{
			run:         sync.NewObj[uuid.UUID](),
			state:       sync.NewObj[State](),
			storyStates: sync.NewMap[storyPart](),
		}
	})
	if !bytes.Equal(d.run.Swap(run).Bytes(), uuid.Nil.Bytes()) {
		d.state.Set(Restarted)
	} else {
		d.state.Set(Started)
	}
	for _, part := range r.s.Parts() {
		state := Waiting
		if s, ok := d.storyStates.Get(part.Id()); ok {
			switch s.State {
			case Started:
				state = Terminated
			case Finished:
				state = Restarted
			}
		}
		d.storyStates.Set(part.Id(), storyPart{
			State: state,
		})
	}
	r.dimensions.Set(dim, d)

	we := event.NewWriteEvent[data](event.Event[data]{
		Type: event.Created,
		Data: data{
			Story:     r.s.Name(),
			Dimension: dim,
			Part:      story.IdStart,
			Run:       run,
			Data: []adapter.Data{
				{
					Type: r.startAdapter.Type(),
					Data: b,
				},
			},
		},
	})
	log.Info("writing new data")
	r.c.Write() <- we
	log.Info("waiting for status of write of new data")
	status := <-we.Done()
	log.Info("wrote new data")
	return status.Error
}

func (r *rr[T]) Read() {
	var part string
	defer func() {
		if e := recover(); e != nil {
			log.WithError(fmt.Errorf("%v", e)).Error("recovered story reader", "story", r.s.Name(), "part", part)
			r.Read()
		}
	}()
	//Stream:
	for e := range r.c.Stream() {
		d, ok := r.dimensions.Get(e.Data.Dimension)
		if !ok {
			log.Error("missing dimention", "story", r.s.Name(), "dimension", e.Data.Dimension)
			continue
		}
		run := d.run.Get()
		log.Trace("read", "current", run.String(), "event", e, "story", r.s.Name(), "part", e.Data.Part, "run", e.Data.Run.String())
		part = e.Data.Part
		if !bytes.Equal(run.Bytes(), e.Data.Run.Bytes()) {
			e.Acc(data{}) //Accing empty event when runid is wrong, sort of a terminate
			continue
		}
		p := r.s.Part(e.Data.Part)
		a := getAdapter(p.Adapter(), r.a)
		if a == nil {
			log.Error("requested adapter not found")
		}
		d.storyStates.Set(p.Id(), storyPart{ //This should be handled by a event or just read from the same event stream.
			State: Started,
		})
		//Since we at this point have no way to stop execution, terminated is not relevant at the moment
		log.Info("executing part", "story", r.s.Name(), "part", p.Id())
		v, err := a.Execute(e.Data.Data)
		log.WithError(err).Info("executed part", "story", r.s.Name(), "part", p.Id())
		if err != nil {
			d.storyStates.Set(p.Id(), storyPart{ //This should be handled by a event
				State: Failed,
			})
			continue
		}
		if v.Data == nil && !adapter.IsNil(a.Type()) {
			log.Error("non nil adapter returned nil data", "adapter", a.Name(), "story", r.s.Name(), "part", p.Id())
			continue
		}
		r.dimensions.CompareAndSwap(e.Data.Dimension, d, func(stored dimension) bool { //This could be a problem, if one task fails when another is read as finished, one of the parts status could get shaddowed
			return bytes.Equal(d.run.Get().Bytes(), stored.run.Get().Bytes())
		})
		/*
			for _, n := range p.Next() {
				we := event.NewWriteEvent[data](event.Event[data]{
					Type: event.Created,
					Data: data{
						Story: r.s.Name(),
						Part:  n,
						Data:  d,
					},
				})
				r.c.Write() <- we
				status := <-we.Done()
				log.WithError(status.Error).Trace("wrote next part", "story", r.s.Name(), "part", p.Id(), "next", n)
				if status.Error != nil {
					continue Stream
				}
			}
		*/
		//Story state at this point is handled by the compleated reader
		e.Acc(data{
			Story:     e.Data.Story,
			Dimension: e.Data.Dimension,
			Part:      e.Data.Part,
			Run:       e.Data.Run,
			//ReadID: e.Data.ReadID,
			Data: []adapter.Data{v},
		})
	}
}

func (r *rr[T]) State() map[string]map[string]State {
	states := make(map[string]map[string]State)
	dimensions := r.dimensions.GetMap()
	for dim, d := range dimensions {
		states[dim] = make(map[string]State)
		m := d.storyStates.GetMap()
		for k, v := range m {
			states[dim][k] = v.State
		}
	}
	return states
}

func getPart(id string, parts []story.Part) story.Part {
	for _, p := range parts {
		if p.Id() != id {
			continue
		}
		return p
	}
	return nil
}

func getAdapter(id string, ads []adapter.Adapter) adapter.Adapter {
	for _, a := range ads {
		if a.Name() != id {
			continue
		}
		return a
	}
	return nil
}

var ErrUnusedAdapter = errors.New("unused adapter provided")
var ErrMissingAdapter = errors.New("at least one adapter is not provided")
