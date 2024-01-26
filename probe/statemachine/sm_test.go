package statemachine

import (
	"context"
	"sync"
	"testing"

	"github.com/cantara/gober/stream"
	"github.com/gofrs/uuid"
)

func inc(i int) int {
	//sbragi.Info("inc", "i", i)
	return i + 1
}
func halv(i int) int {
	//sbragi.Info("halv", "i", i)
	return i / 2
}

func TestBase(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sm, err := New[int]("TestStateMaching_"+uuid.Must(uuid.NewV7()).String(), stream.StaticProvider("aPSIX6K3yw6cAWDQHGPjmhuOswuRibjyLLnd91ojdK0="), "inc", ctx)
	if err != nil {
		t.Fatal(err)
		return
	}
	sm.Func("inc", func(i int) (State[int], error) {
		i = inc(i)
		state := "inc"
		if 100 == i {
			state = "halv"
		}
		return State[int]{
			Data:  i,
			State: state,
		}, nil
	})
	var wg sync.WaitGroup
	sm.Func("halv", func(i int) (_ State[int], _ error) {
		i = halv(i)
		if 50 != i {
			t.Fatalf("half of b.N != i, %d != %d", 50, i)
		}
		wg.Done()
		return
	})
	go sm.Run(ctx)

	wg.Add(1)
	sm.Start(0)
	wg.Wait()
}
func TestContinue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	sn := "TestStateMaching_" + uuid.Must(uuid.NewV7()).String()
	sm, err := New[int](sn, stream.StaticProvider("aPSIX6K3yw6cAWDQHGPjmhuOswuRibjyLLnd91ojdK0="), "inc", ctx)
	if err != nil {
		t.Fatal(err)
		return
	}
	var wg sync.WaitGroup
	sm.Func("inc", func(i int) (State[int], error) {
		i = inc(i)
		state := "inc"
		if 25 == i {
			cancel()
			wg.Done()
		}
		return State[int]{
			Data:  i,
			State: state,
		}, nil
	})
	go sm.Run(ctx)

	wg.Add(1)
	sm.Start(0)
	wg.Wait()

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	sm, err = New[int](sn, stream.StaticProvider("aPSIX6K3yw6cAWDQHGPjmhuOswuRibjyLLnd91ojdK0="), "inc", ctx)
	if err != nil {
		t.Fatal(err)
		return
	}
	sm.Func("inc", func(i int) (State[int], error) {
		i = inc(i)
		state := "inc"
		if 100 == i {
			state = "halv"
		}
		return State[int]{
			Data:  i,
			State: state,
		}, nil
	})
	sm.Func("halv", func(i int) (_ State[int], _ error) {
		i = halv(i)
		if 50 != i {
			t.Fatalf("half of b.N != i, %d != %d", 50, i)
		}
		wg.Done()
		return
	})
	go sm.Run(ctx)

	wg.Add(1)
	wg.Wait()
}

func BenchmarkState(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sm, err := New[int]("BenchmarkStateMaching_"+uuid.Must(uuid.NewV7()).String(), stream.StaticProvider("aPSIX6K3yw6cAWDQHGPjmhuOswuRibjyLLnd91ojdK0="), "inc", ctx)
	if err != nil {
		b.Fatal(err)
		return
	}
	sm.Func("inc", func(i int) (State[int], error) {
		i = inc(i)
		state := "inc"
		if b.N == i {
			state = "halv"
		}
		return State[int]{
			Data:  i,
			State: state,
		}, nil
	})
	var wg sync.WaitGroup
	sm.Func("halv", func(i int) (_ State[int], _ error) {
		i = halv(i)
		if b.N/2 != i {
			b.Fatalf("half of b.N != i, %d != %d", b.N/2, i)
		}
		wg.Done()
		return
	})
	go sm.Run(ctx)

	wg.Add(1)
	b.ResetTimer()
	sm.Start(0)
	wg.Wait()
}
