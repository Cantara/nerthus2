package executor

type Func func(chan<- Func)

type Executor interface {
	Run()
	Add(f Func)
}

type executor struct {
	funcs chan Func
}

const buffer = 4096

func (e *executor) Run() {
	for f := range e.funcs {
		f(e.funcs)
	}
}

func (e *executor) Add(f Func) {
	e.funcs <- f
}

func NewExecutor() executor {
	return executor{
		funcs: make(chan Func, buffer),
	}
}
