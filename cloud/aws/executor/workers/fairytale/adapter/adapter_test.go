package adapter

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	log "github.com/cantara/bragi/sbragi"
)

/*
type inn struct {
	Name string `json:"name"`
}
*/

type inn struct {
	Name string `json:"name"`
}

func newOf(v any) any {
	switch v.(type) {
	case inn:
		return &inn{}
	default:
		return v
	}
}

func toNames(s []Value) []string {
	o := make([]string, len(s))
	for i, v := range s {
		o[i] = Name(v)
	}
	return o
}

var AdapterFingerprintInn = New[inn]("inn")
var AdapterFingerprint1 = New[string]("test")

func TestAdapter(t *testing.T) {
	a := AdapterFingerprint1.Adapter(func(s []Value) (string, error) {
		log.Info("func", "s", toNames(s))
		//return s[0].(*inn).Name, nil
		arg, ok := AdapterFingerprintInn.TryValue(s[0])
		if !ok {
			return "", errors.New("argument was incorrect type")
		}
		return arg.Name, nil
	}, AdapterFingerprintInn)

	/*New[string]("test", func(s []any) (string, error) {
		log.Info("func", "s", toNames(s))
		//return s[0].(*inn).Name, nil
		arg, ok := Value[inn](s[0])
		if !ok {
			return "", errors.New("argument was incorrect type")
		}
		return arg.Name, nil
	}, GenNew[inn]())*/
	o, err := a.Execute([]Data{
		{
			Type: AdapterFingerprintInn.Type(),
			Data: []byte("{\"name\":\"sindre\"}"),
		},
	})
	//o, err := a.Execute([][]byte{[]byte("{\"name\":9}")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(o.Data), "sindre") {
		t.Error("sindre is not contained in returned value")
	}
	fmt.Println(o.Type, string(o.Data))
}

func BenchmarkAdapter(b *testing.B) {
	a := New[string]("bench").Adapter(func(s []Value) (string, error) {
		return AdapterFingerprintInn.Value(s[0]).Name, nil
	}, AdapterFingerprintInn)
	innData := []Data{
		{
			Type: AdapterFingerprintInn.Type(),
			Data: []byte("{\"name\":\"sindre\"}"),
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.Execute(innData)
		if err != nil {
			b.Fatal(err)
		}
	}
}
