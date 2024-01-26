package statemachine

import (
	"context"

	"github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/stream"
	"github.com/cantara/gober/stream/consumer"
	"github.com/cantara/gober/stream/event"
	"github.com/cantara/gober/stream/event/store/ondisk"
)

type State[T any] struct {
	State string
	Data  T
}
type Fn[T any] func(T) (State[T], error)
type StateMachine[T any] interface {
	Func(name string, fn Fn[T])
	Run(context.Context)
	Start(data T)
}
type stateMachine[T any] struct {
	funcs map[string]Fn[T]
	start string
	c     consumer.Consumer[State[T]]
	s     <-chan event.ReadEventWAcc[State[T]]
	ctx   context.Context
}

func New[T any](streamName string, ckp stream.CryptoKeyProvider, startState string, ctx context.Context) (StateMachine[T], error) {
	strm, err := ondisk.Init(streamName, ctx)
	if sbragi.WithError(err).Trace("creating streame") {
		return nil, err
	}
	c, err := consumer.New[State[T]](strm, ckp, ctx)
	if sbragi.WithError(err).Trace("creating consumer") {
		return nil, err
	}
	s, err := c.StreamPers(event.AllTypes(), stream.ReadAll(), ctx)
	return &stateMachine[T]{
		funcs: make(map[string]Fn[T]),
		start: startState,
		c:     c,
		s:     s,
		ctx:   ctx,
	}, nil
}

func (s *stateMachine[T]) Func(name string, fn Fn[T]) {
	s.funcs[name] = fn
}

func (s *stateMachine[T]) event(next State[T]) event.WriteEventReadStatus[State[T]] {
	return event.NewWrite(event.Created, next,
		event.Metadata{
			Version:  "0.0.1",
			DataType: "state",
			//Key:      crypto.GenRandBase32String(32), //crypto.SimpleHash(data.Metadata.Id),
		},
	)
}

func (s *stateMachine[T]) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			sbragi.Trace("run context done")
			return
		case <-s.ctx.Done():
			sbragi.Trace("state context done")
			return
		case e := <-s.s:
			f, ok := s.funcs[e.Data.State]
			if !ok {
				sbragi.Error("state function not found", "state", e.Data.State)
				continue
			}
			next, err := f(e.Data.Data)
			if sbragi.WithError(err).Trace("state function executed", "state", e.Data.State, "next", next.State) {
				continue
			}
			if next.State == "" {
				e.Acc()
				continue
			}
			we := s.event(next)
			select {
			case <-ctx.Done():
				sbragi.Trace("run context done")
				return
			case <-s.ctx.Done():
				sbragi.Trace("state context done")
				return
			case s.c.Write() <- we:
				/*
					select {
					case <-ctx.Done():
						sbragi.Trace("run context done")
						return
					case <-s.ctx.Done():
						sbragi.Trace("state context done")
						return
					case status := <-we.Done():
						sbragi.Trace("DONE")
						sbragi.WithError(status.Error).Trace("next status written")
					}
				*/
			}
			e.Acc()
		}
	}
}

func (s *stateMachine[T]) Start(data T) {
	we := s.event(State[T]{
		State: s.start,
		Data:  data,
	})

	select {
	case <-s.ctx.Done():
		sbragi.Trace("state context done")
		return
	case s.c.Write() <- we:
		select {
		case <-s.ctx.Done():
			sbragi.Trace("state context done")
			return
		case status := <-we.Done():
			sbragi.WithError(status.Error).Trace("start status written")
		}
	}
}
