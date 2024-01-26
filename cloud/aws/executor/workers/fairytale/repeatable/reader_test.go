package repeatable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/types"
	"strings"
	"sync"
	"testing"
	"time"

	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/consensus"
	"github.com/cantara/gober/discovery/local"
	"github.com/cantara/gober/stream"
	"github.com/cantara/gober/stream/event/store/inmemory"
	"github.com/cantara/gober/stream/event/store/ondisk"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/story"
	"github.com/gofrs/uuid"
)

var ctxGlobal context.Context
var ctxGlobalCancel context.CancelFunc
var STREAM_NAME = "TestFairytale_" + uuid.Must(uuid.NewV7()).String()
var testCryptKey = log.RedactedString("aPSIX6K3yw6cAWDQHGPjmhuOswuRibjyLLnd91ojdK0=")

var r Reader
var wg sync.WaitGroup

func TestNew(t *testing.T) {
	wtgf := adapter.New[string]("WriteToGraph")
	wtg := wtgf.Adapter(func(a []adapter.Value) (string, error) {
		log.Info("wtg", "a", a)
		return "wtg", nil
	})
	wtrf := adapter.New[string]("WriteToRDBMS")
	wtbf := adapter.New[string]("WriteToBarf")
	wtb := wtbf.Adapter(func(a []adapter.Value) (string, error) {
		log.Info("wtb", "a", a)
		s := make([]string, 2)
		s[0] = wtgf.Value(a[0])
		s[1] = wtrf.Value(a[1])
		log.Info("wtb", "s", s)
		return strings.Join(s, "_") + "_wtb", nil
	}, wtgf, wtrf)
	wtr := wtrf.Adapter(func(a []adapter.Value) (string, error) {
		log.Info("wtr", "a", a)
		return /*s[0] + "_*/ "wtr", nil
	})
	failCount := 0
	wtosf := adapter.New[string]("WriteToObjectStore")
	wtpsf := adapter.New[string]("PublishToPubSub")
	wtps := wtpsf.Adapter(func(a []adapter.Value) (string, error) {
		s := make([]string, 3)
		s[0] = wtbf.Value(a[0])
		s[1] = wtrf.Value(a[1])
		s[2] = wtosf.Value(a[2])
		log.Info("wtps", "s", s)
		if failCount < 2 {
			failCount++
			return "", errors.New("dummy error")
		}
		return strings.Join(s, "_") + "_wtps", nil
	}, wtbf, wtrf, wtosf)
	wtos := wtosf.Adapter(func(a []adapter.Value) (string, error) {
		log.Info("wtos", "a", a)
		return /*s[0] + "_*/ "wtos", nil
	})
	s, err := story.Start("test story").
		LinkTo("rdbms", "graph", "objectstore").
		Id("graph").Adapter(wtg.Name()).LinkTo("barf").
		Id("barf").Adapter(wtb.Name()).LinkTo("pubsub").
		Id("rdbms").Adapter(wtr.Name()).LinkTo("pubsub", "barf").
		Id("pubsub").Adapter(wtps.Name()).LinkToEnd().
		Id("objectstore").Adapter(wtos.Name()).LinkTo("pubsub").
		End()
	if err != nil {
		t.Fatal(err)
	}
	ctxGlobal, ctxGlobalCancel = context.WithCancel(context.Background())
	store, err := ondisk.Init(STREAM_NAME, ctxGlobal)
	if err != nil {
		t.Fatal(err)
	}
	token := "someTestToken"
	p, err := consensus.Init(3134, token, local.New())
	if err != nil {
		t.Fatal(err)
	}
	_, err = New[types.Nil](store, p.AddTopic, stream.StaticProvider(testCryptKey), time.Second, s, ctxGlobal)
	if !errors.Is(err, ErrMissingAdapter) {
		t.Fatal("did not get missing adapter error when no adapters were provided")
		return
	}
	dumy := adapter.New[string]("dumy").Adapter(func(a []adapter.Value) (string, error) {
		return a[0].(string) + "dumy", nil
	})
	_, err = New[types.Nil](store, p.AddTopic, stream.StaticProvider(testCryptKey), time.Second, s, ctxGlobal, wtb, wtg, wtr, wtps, wtos, dumy)
	if !errors.Is(err, ErrUnusedAdapter) {
		t.Fatalf("did not get unused adapter error when dumy adapter was provided. err:%v", err)
		return
	}
	_, err = New[types.Nil](store, p.AddTopic, stream.StaticProvider(testCryptKey), time.Second, s, ctxGlobal, wtb, wtg, wtr, wtps, wtos, wtos)
	if !errors.Is(err, ErrUnusedAdapter) {
		t.Fatal("did not get unused adapter error when duplicate adapter was provided")
		return
	}
	r, err = New[types.Nil](store, p.AddTopic, stream.StaticProvider(testCryptKey), time.Second, s, ctxGlobal, wtb, wtg, wtr, wtps, wtos, adapter.New[string](story.AdapterEnd).Adapter(func(a []adapter.Value) (string, error) {
		log.Info("end", "a", a)
		str := wtpsf.Value(a[0])
		log.Info("end", "s", str)
		if !strings.Contains(str, "wtb") {
			log.Fatal("missing wtb")
		}
		if str != "wtg_wtr_wtb_wtr_wtos_wtps" {
			log.Fatal("did not get correct string", "got", str)
		}
		wg.Done()
		return str, nil
	}, wtpsf))
	if err != nil {
		t.Fatal(err)
	}
	go p.Run()
}

func TestRead(t *testing.T) {
	r.New("read", nil)
	wg.Add(1)
	go r.Read()
	wg.Wait()
}

/*
type Int struct {
	Num int64 `json:"num"`
}
*/

func TestCrazyTown(t *testing.T) {
	var r2 Reader
	dim := "ct"
	start := adapter.New[int](story.AdapterStart)
	wtgf := adapter.New[int]("WriteToGraph")
	wtg := wtgf.Adapter(func(a []adapter.Value) (int, error) {
		log.Info("wtg", "a", a)
		num := start.Value(a[0])
		num++
		log.Info("wtg", "num", num)
		return /*s[0] + "_ "wtg"*/ num, nil
	}, start)
	wtrf := adapter.New[string]("WriteToRDBMS")
	wtr := wtrf.Adapter(func(a []adapter.Value) (string, error) {
		log.Info("wtr", "a", a)
		return /*s[0] + "_*/ "wtr", nil
	})
	wtbf := adapter.New[string]("WriteToBarf")
	wtb := wtbf.Adapter(func(a []adapter.Value) (string, error) {
		log.Info("wtb", "a", a)
		str := wtrf.Value(a[0])
		num := wtgf.Value(a[1])
		log.Info("wtb", "s", str, "n", num)
		if num < 10 {
			b, err := json.Marshal(num)
			if err != nil {
				return "", err
			}
			states := r2.State()
			fmt.Println("States running:")
			for k, v := range states[dim] {
				fmt.Printf("%s: %s\n", k, v)
				var e State
				switch k {
				case story.IdStart:
					fallthrough
				case "rdbms":
					fallthrough
				case "graph":
					e = Finished
				case "barf":
					e = Started
				case story.IdEnd:
					e = Waiting
				}
				if v == e {
					continue
				}
				t.Fatalf("wrong status, id=%s, status=%s, expected=%s", k, v, e)
			}
			r2.New(dim, b)
			states = r2.State()
			fmt.Println("States restarted:")
			for k, v := range states[dim] {
				fmt.Printf("%s: %s\n", k, v)
				var e State
				switch k {
				case story.IdStart:
					fallthrough
				case "rdbms":
					fallthrough
				case "graph":
					e = Restarted
				case "barf":
					e = Terminated
				case story.IdEnd:
					e = Waiting
				}
				if v == e {
					continue
				}
				t.Fatalf("wrong status, id=%s, status=%s, expected=%s", k, v, e)
			}
			return "", fmt.Errorf("num too small")
		}
		return fmt.Sprintf("%s_%d_wtb", str, num), nil
	}, wtrf, wtgf)
	s, err := story.Start("test story 2").
		LinkTo("rdbms", "graph").
		Id("graph").Adapter(wtg.Name()).LinkTo("barf").
		Id("rdbms").Adapter(wtr.Name()).LinkTo("barf").
		Id("barf").Adapter(wtb.Name()).LinkToEnd().
		End()
	if err != nil {
		t.Fatal(err)
	}
	ctxGlobal, ctxGlobalCancel = context.WithCancel(context.Background())
	store, err := inmemory.Init(STREAM_NAME, ctxGlobal)
	if err != nil {
		t.Fatal(err)
	}
	token := "someTestToken"
	p, err := consensus.Init(3135, token, local.New())
	if err != nil {
		t.Fatal(err)
	}
	r2, err = New[int](store, p.AddTopic, stream.StaticProvider(testCryptKey), time.Second, s, ctxGlobal, start.Adapter(func(a []adapter.Value) (int, error) {
		log.Info("start", "a", a)
		return start.Value(a[0]), nil
	}, start), wtb, wtg, wtr, adapter.New[string](story.AdapterEnd).Adapter(func(a []adapter.Value) (string, error) {
		str := wtbf.Value(a[0]) //*a[0].(*string)
		//str := strings.Join(s, "_")
		log.Info("end", "s", str)
		if str != "wtr_10_wtb" {
			t.Errorf("end result incorrect: \"%s\" != \"wtr_10_wtb\"", str)
		}
		wg.Done()
		return str, nil
	}, wtbf))
	if err != nil {
		t.Fatal(err)
	}
	go p.Run()
	r2.New(dim, []byte("8"))
	wg.Add(1)
	go r2.Read()
	wg.Wait()
}

func BenchmarkCrazy(b *testing.B) {
	var wg sync.WaitGroup
	wtgf := adapter.New[int]("WriteToGraph")
	wtg := wtgf.Adapter(func(a []adapter.Value) (int, error) {
		log.Info("wtg", "a", a)
		return /*s[0] + "_ "wtg"*/ 9, nil
	})
	wtrf := adapter.New[string]("WriteToRDBMS")
	wtr := wtrf.Adapter(func(a []adapter.Value) (string, error) {
		log.Info("wtr", "a", a)
		return /*s[0] + "_*/ "wtr", nil
	})
	wtbf := adapter.New[string]("WriteToBarf")
	wtb := wtbf.Adapter(func(a []adapter.Value) (string, error) {
		log.Info("wtb", "a", a)
		str := wtrf.Value(a[0])
		num := wtgf.Value(a[1])
		log.Info("wtb", "s", str, "n", num)
		return fmt.Sprintf("%s_%d_wtb", str, num), nil
	}, wtrf, wtgf)
	s, err := story.Start(fmt.Sprintf("test story %d", 2+b.N)).
		LinkTo("rdbms", "graph").
		Id("graph").Adapter(wtg.Name()).LinkTo("barf").
		Id("rdbms").Adapter(wtr.Name()).LinkTo("barf").
		Id("barf").Adapter(wtb.Name()).LinkToEnd().
		End()
	if err != nil {
		b.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := ondisk.Init(STREAM_NAME, ctx)
	if err != nil {
		b.Fatal(err)
	}
	token := "someTestToken"
	p, err := consensus.Init(uint16(3236+b.N), token, local.New())
	if err != nil {
		b.Fatal(err)
	}
	r2, err := New[types.Nil](store, p.AddTopic, stream.StaticProvider(testCryptKey), time.Second, s, ctxGlobal, wtb, wtg, wtr, adapter.New[string](story.AdapterEnd).Adapter(func(a []adapter.Value) (string, error) {
		str := wtbf.Value(a[0])
		//str := strings.Join(s, "_")
		log.Info("end", "s", str)
		if str != "wtr_9_wtb" {
			b.Errorf("end result incorrect: \"%s\" != \"wtr_9_wtb\"", str)
		}
		wg.Done()
		return str, nil
	}, wtbf))
	if err != nil {
		b.Fatal(err)
	}
	go p.Run()
	for i := 0; i < b.N; i++ {
		r2.New("bench", nil)
		wg.Add(1)
	}
	b.ResetTimer()
	go r2.Read()
	wg.Wait()
}
