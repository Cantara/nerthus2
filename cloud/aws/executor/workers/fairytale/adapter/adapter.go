package adapter

import (
	"errors"
	"fmt"

	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/sync"
	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
)

var json = jsoniter.ConfigDefault

const (
	Start = "Start"
	End   = "End"
)

type NilType struct{}

//const NilName = "struct {}" // = Name(*new(NilType))

func IsNil(t string) bool {
	return t == "struct {}" || t == "adapter.NilType"
}

type fingerprint[T any] struct {
	name string
	ot   string
}
type adapter[T any] struct {
	f    ExecFunc[T]
	reqs []Fingerprint
	fingerprint[T]
}

var registered = sync.NewMap[fingerprint[any]]()

type ExecFunc[T any] func([]Value) (T, error)
type Value any

type Fingerprint interface {
	Name() string
	Type() string
	New() any
}

type FingerprintBase[T any] interface {
	Name() string
	Type() string
	Adapter(ExecFunc[T], ...Fingerprint) Adapter
	New() any
	TryValue(Value) (t T, ok bool)
	Value(Value) (t T)
}

type Data struct {
	Type string `json:"type"`
	Data []byte `json:"data,omitempty"`
}

type Adapter interface {
	Name() string
	Type() string
	Execute([]Data) (Data, error)
	Reqs() []string
}

func (af fingerprint[T]) Adapter(f ExecFunc[T], reqs ...Fingerprint) Adapter {
	return &adapter[T]{
		f:           f,
		reqs:        reqs,
		fingerprint: af,
	}
}

func (af fingerprint[T]) New() any {
	var t T
	return &t
}

func (af fingerprint[T]) TryValue(a Value) (t T, ok bool) {
	var v *T
	v, ok = a.(*T)
	if !ok {
		return
	}
	t = *v
	return
}

func (af fingerprint[T]) Value(a Value) (t T) {
	return *a.(*T)
}

func (af fingerprint[T]) Name() string {
	return af.name
}

func (af fingerprint[T]) Type() string {
	return af.ot
}

func New[T any](name string) FingerprintBase[T] {
	var t T
	log.Info("check adapter creation", "t", t, "name", name, "Name(t)", Name(t), "new", Name(*new(T)), "fmt", fmt.Sprintf("%T", *new(T)))
	return fingerprint[T]{
		name: name,
		ot:   Name(t),
	}
}

func (a *adapter[T]) Name() string {
	return a.name
}

func (a adapter[T]) Type() string {
	return a.ot
}

func (a *adapter[T]) Reqs() []string {
	s := make([]string, len(a.reqs))
	for i, req := range a.reqs {
		s[i] = req.Name()
	}
	return s
}

func (a *adapter[T]) Execute(ds []Data) (o Data, err error) {
	//var t []any
	//ts := []any{inn{}} //a.ts
	ts := make([]Value, len(a.reqs))
	//if len(b) > 0 {
	used := make([]bool, len(ds))
types:
	for j, req := range a.reqs {
		//t := reflect.Indirect(reflect2.TypeOf(ts[j]).New())
		//t := reflect2.TypeOf(a.ts[j]).UnsafeNew()
		//var t ts[j].(type)
		for i, d := range ds {
			if used[i] {
				continue
			}
			/*
				Do not believe this is needed.
				if req.Type() == "" { //FIXME: This should be ignorable.
					var v map[string]any
					err = json.Unmarshal(d.Data, &v)
					if log.WithoutEscalation().WithError(err).Trace("unmarshaling", "t", v, "b", string(d.Data), "it", Name(v), "ot", a.ot) {
						continue
					}
					used[i] = true
					ts[j] = v
					continue types
				}
			*/
			if req.Type() == "" {
				log.Warning("type was empty, not expected", "type", d.Type, "req", req.Type())
				continue
			}
			log.Info("data", "type", d.Type, "req", req.Type())
			if d.Type != req.Type() {
				continue
			}
			if IsNil(d.Type) { //d.Type == NilName {
				used[i] = true
				continue types
			}
			t := req.New()
			err = json.Unmarshal(d.Data, t)
			log.WithError(err).Trace("unmarshaling", "t", t, "b", string(d.Data), "it", Name(t), "ot", a.ot)
			if err != nil {
				continue
			}
			used[i] = true
			ts[j] = t
			continue types
		}
		err = errors.New("not all parameters were provided")
		return
	}
	if a.name != End {
		for i, ok := range used {
			if !ok {
				log.Info("data1", "type", a.Type(), "nil?", IsNil(a.Type()))
				log.Info("data2", "type", ds[i].Type, "nil?", IsNil(ds[i].Type))
			}
			if !ok && !IsNil(ds[i].Type) {
				err = fmt.Errorf("not all data was used data=%s type=%s ds=%d ts=%d", string(ds[i].Data), ds[i].Type, len(ds), len(ts))
				return
			}
		}
	}
	v, err := a.f(ts)
	if err != nil {
		return
	}
	log.Info("adapter check", "name", Name(v), "type", a.Type(), "isNil", IsNil(a.Type()))
	var b []byte
	if !IsNil(a.Type()) {
		b, err = json.Marshal(v)
		if err != nil {
			return
		}
	}
	return Data{
		Type: a.Type(),
		Data: b,
	}, nil
}

func Name(v any) string {
	rt := reflect2.TypeOf(v)
	if rt == nil {
		return ""
	}
	return rt.String()
}
