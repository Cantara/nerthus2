package saga

import (
	"reflect"
	"sync"
	"sync/atomic"

	log "github.com/cantara/bragi/sbragi"
)

type SagaExecuteFunc[RT any] func() (RT, error)
type Requireing[T any] func(T) Executable

type Event[RT any] struct {
	Func        SagaExecuteFunc[RT]
	mandateList []Requireing[RT]
	reqList     []bool

	lock sync.Mutex
}

func (e *Event[RT]) Mandates(r Requireing[RT]) {
	e.mandateList = append(e.mandateList, r)
}

func (e *Event[RT]) Distribute(r RT) (n []Executable) {
	log.Info("Distributing", "list", len(e.mandateList))
	for _, req := range e.mandateList {
		f := req(r)
		if f == nil {
			continue
		}
		n = append(n, f)
	}
	return
}

func (e *Event[RT]) Execute() (n []Executable, err error) {
	var rt RT
	rt, err = e.Func()
	if err != nil {
		return
	}
	n = e.Distribute(rt)
	return
}

func Mandatabale[T, ST any](e *Event[ST], f func(t T)) func(T) Executable {
	e.lock.Lock()
	defer e.lock.Unlock()
	i := len(e.reqList)
	e.reqList = append(e.reqList, false)
	return func(t T) Executable {
		f(t)
		e.lock.Lock()
		defer e.lock.Unlock()
		e.reqList[i] = true
		for _, r := range e.reqList {
			if !r {
				return nil
			}
		}
		return e
	}
}

type Executable interface {
	Execute() ([]Executable, error)
}

type Saga struct {
	events []Executable

	ends     atomic.Uint32
	finished atomic.Uint32
	wg       sync.WaitGroup
}

func (s *Saga) Event(e Executable) {
	s.events = append(s.events, e)
}

func (s *Saga) Execute(c chan<- Executable) {
	s.wg.Add(1)
	log.Info("Executing saga")
	for _, e := range s.events {
		c <- e
	}
	s.wg.Wait()
}

func (s *Saga) End(any) Executable {
	s.finished.Add(1)
	log.Info("testing finished", "ends", s.ends.Load(), "finished", s.finished.Load())
	if s.ends.Load() != s.finished.Load() {
		return nil
	}
	return &Event[any]{Func: func() (any, error) {
		defer s.wg.Done()
		log.Info("Saga ended")
		return nil, nil
	}}
}

func End[T any](s *Saga) Requireing[T] {
	s.ends.Add(1)
	return func(T) Executable {
		return s.End(nil)
	}
}

func Worker(c chan Executable) {
	for e := range c {
		ns, err := e.Execute()
		if err != nil {
			log.WithError(err).Error("while executing", "name", reflect.ValueOf(e))
			c <- e
		}
		log.Info("Adding events that required finished event", "new", len(ns))
		for _, n := range ns {
			c <- n
		}
	}
}
