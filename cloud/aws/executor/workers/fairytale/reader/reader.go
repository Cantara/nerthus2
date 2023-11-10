package reader

import (
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
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/story"
	"github.com/gofrs/uuid"
)

// var StartInputFingerprint = adapter.New[struct{}](story.AdapterStart)
var StartAdapterFingerprint = adapter.New[struct{}](story.AdapterStart)

var StartAdapter = StartAdapterFingerprint.Adapter(func(n []adapter.Value) (struct{}, error) {
	log.Info("start", "n", n)
	return struct{}{}, nil
})

/*
func startFunc[T any](b interface{ Value(adapter.Value) (t T) }) func(n []adapter.Value) (T, error) {
	var t T
	switch interface{}(t).(type) {
	case types.Nil:
		return func(n []adapter.Value) (t T, err error) {
			log.Info("start", "n", n)
			return
		}
	default:
		return func(n []adapter.Value) (T, error) {
			log.Info("start", "n", n)
			return b.Value(n[0]), nil
		}
	}
}
*/

var anyAdapterFingerprint = adapter.New[any]("any")
var EndAdapter = adapter.New[[]adapter.Value](story.AdapterEnd).Adapter(func(n []adapter.Value) ([]adapter.Value, error) { return n, nil }, anyAdapterFingerprint)

type Reader interface {
	New([]byte) error
	Read()
}

type reader[T any] struct {
	c    competing.Consumer[data]
	s    story.Story
	a    []adapter.Adapter
	cons consensus.Consensus

	startAdapter adapter.Adapter
}

type data struct {
	Story  string
	ReadID uuid.UUID
	Part   string
	Data   []adapter.Data
}

func New[T any](strm stream.Stream, consBuilder consensus.ConsBuilderFunc, ckp stream.CryptoKeyProvider, timeout time.Duration, s story.Story, ctx context.Context, adapters ...adapter.Adapter) (Reader, error) {
	//StartFingerprint := adapter.New[T](story.AdapterStart)
	//StartAdapter := StartFingerprint.Adapter(startFunc[T](StartFingerprint))
	if getAdapter(story.AdapterStart, adapters) == nil {
		adapters = append(adapters, StartAdapter)
	}
	sa := getAdapter(story.AdapterStart, adapters)
	if getAdapter(story.AdapterEnd, adapters) == nil {
		adapters = append(adapters, EndAdapter)
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

	go func() { //Read completed chan and keep a record of all completes and create next events that can now be created
		comp := sync.NewMap[sync.Map[data]]()
		for e := range c.Completed() {
			r, _ := comp.GetOrInit(e.Data.ReadID.String(), sync.NewMap[data])
			p := s.Part(e.Data.Part)
			r.Set(p.Id(), e.Data)
		NextPart:
			for _, n := range p.Next() {
				np := s.Part(n)
				//var buf bytes.Buffer
				//buf.WriteByte('[')
				td := make([]adapter.Data, len(np.Prev()))
				for i, npr := range np.Prev() {
					d, ok := r.Get(npr)
					if !ok {
						log.Info("required part not completed", "part", npr)
						continue NextPart
					}
					td[i] = d.Data[0]
					/*
						if i != 0 {
							buf.WriteByte(',')
						}
						buf.Write(d.Data)
					*/
				}
				//buf.WriteByte(']')
				we := event.NewWriteEvent[data](event.Event[data]{
					Type: event.Created,
					Data: data{
						Story:  s.Name(),
						Part:   n,
						ReadID: e.Data.ReadID, //To me it seems like not having this would create issues, however, from my test it does not. Might need a longer story
						Data:   td,            //buf.Bytes(),   //This should become a array of all datas that is required
					},
				})
				consId := fmt.Sprintf("%s_%s", np.Id(), e.Data.ReadID.String())
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

	return &reader[T]{
		c:            c,
		s:            s,
		a:            adapters,
		cons:         cons,
		startAdapter: sa,
	}, nil
}

func (r *reader[T]) New(d []byte) error {
	we := event.NewWriteEvent[data](event.Event[data]{
		Type: event.Created,
		Data: data{
			Story:  r.s.Name(),
			Part:   story.IdStart,
			ReadID: uuid.Must(uuid.NewV7()),
			Data: []adapter.Data{
				{
					Type: r.startAdapter.Type(),
					Data: d,
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

func (r *reader[T]) Read() {
	defer func() {
		if e := recover(); e != nil {
			log.WithError(fmt.Errorf("%v", e)).Error("recovered story reader")
			r.Read()
		}
	}()
	//Stream:
	for e := range r.c.Stream() {
		log.Trace("read", "event", e, "story", r.s.Name(), "part", e.Data.Part)
		p := r.s.Part(e.Data.Part)
		a := getAdapter(p.Adapter(), r.a)
		if a == nil {
			log.Error("requested adapter not found")
		}
		d, err := a.Execute(e.Data.Data)
		log.WithError(err).Trace("executed part", "story", r.s.Name(), "part", p.Id())
		if err != nil {
			continue
		}
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
		e.Acc(data{
			Story:  e.Data.Story,
			Part:   e.Data.Part,
			ReadID: e.Data.ReadID,
			Data:   []adapter.Data{d},
		})
	}
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
