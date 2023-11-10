package story

import (
	"errors"

	log "github.com/cantara/bragi/sbragi"
)

const (
	IdStart      = "S"
	IdEnd        = "E"
	AdapterStart = "SagaStart"
	AdapterEnd   = "SagaEnd"
)

type Story interface {
	Name() string
	Parts() []Part
	Part(string) Part
}

type story struct {
	name  string
	nodes []part
}

func (s story) Name() string {
	return s.name
}

func (s story) Part(id string) Part {
	for _, n := range s.nodes {
		if n.id != id {
			continue
		}
		return n
	}
	return part{}
}

func (s story) Parts() (p []Part) {
	for _, n := range s.nodes {
		p = append(p, n)
	}
	return
}

type node struct {
	id  string
	req func()
}

func Start(name string) *OutgoingBuilder {
	s := StoryBuilder{name: name}
	return s.Id(IdStart).Adapter(AdapterStart)
}

type StoryBuilder struct {
	name  string
	nodes []part //State
}

func (b *StoryBuilder) Id(id string) *AdapterBuilder {
	i := containsNode(id, b.nodes)
	if i >= 0 {
		return &AdapterBuilder{
			ns: &b.nodes[i],
			sb: b,
		}
	}
	ns := part{id: id}
	b.nodes = append(b.nodes, ns)
	return &AdapterBuilder{
		ns: &b.nodes[len(b.nodes)-1],
		sb: b,
	}
}

func (b *StoryBuilder) End() (s Story, err error) {
	b.Id(IdEnd).Adapter(AdapterEnd)
	b.buildInnLinks()
	err = b.sortGraph()
	if err != nil {
		return
	}
	err = b.validateAsyclic(0, make([]bool, len(b.nodes)))
	if err != nil {
		return
	}
	//b.validateReachable()
	s = story{name: b.name, nodes: b.nodes}
	return
}

func (b *StoryBuilder) buildInnLinks() {
	for i, n := range b.nodes {
		for _, v := range b.nodes {
			if contains(n.id, v.outgoing) < 0 {
				continue
			}
			b.nodes[i].Inngoing(v.id)
		}
	}
}

func (b *StoryBuilder) validateAsyclic(i int, seen []bool) error {
	if seen[i] {
		return ErrSyclicGraph
	}
	seen[i] = true
	for _, out := range b.nodes[i].outgoing {
		s := make([]bool, len(seen))
		copy(s, seen)
		err := b.validateAsyclic(containsNode(out, b.nodes), s)
		if err != nil {
			return err
		}
	}
	/*
		for i, n := range b.nodes {
			seen := make([]bool, len(b.nodes))
			seen[i] = true
			s := stack.New[string]()
			for _, out := range n.outgoing {
				s.Push(out)
			}
			for out, ok := s.Pop(); ok; {
				outI := containsNode(out, b.nodes)
				if seen[outI] {
					return fmt.Errorf("syclic graph")
				}
				seen[outI] = true
				for _, out := range b.nodes[outI].outgoing {
					s.Push(out)
				}
			}
		}
	*/
	return nil
}

/*
type ErrMissingNode struct {
	id string
}

func (e ErrMissingNode) Error() string {
	return fmt.Sprintf("Missing node in graph, id=%s", e.id)
}
*/

func (b *StoryBuilder) sortGraph() error {
	i, cur := 1, 0
	nodes := make([]part, len(b.nodes))
	nodes[cur] = b.nodes[containsNode(IdStart, b.nodes)]
	for i < len(nodes) && cur < len(nodes)-1 {
		for _, out := range nodes[cur].outgoing {
			//TODO: Shuffle if outgoing is already in nodes
			//	might need to consider incomming aswell
			if out == IdEnd {
				continue
			}

			//log.Info("outgoing", "out", out, "cur", nodes[cur].id)
			if outI := containsNode(out, nodes[:cur]); outI >= 0 { //This is probably wrong
				//log.Info("contains", "out", out, "cur", nodes[cur].id, "i", i, "outI", outI)
				n := nodes[outI]
				nodes[outI] = nodes[cur]
				nodes[cur] = n
				//nodes[i] = nodes[outI]
				//nodes[outI] = b.nodes[containsNode(out, b.nodes)]
				//i++
				continue
			}
			if containsNode(out, nodes[cur:]) >= 0 { //This is probably wrong
				continue
			}

			ni := containsNode(out, b.nodes)
			if ni < 0 {
				return ErrMissingNode //{id: out}
				/*
					log.Warning("missing node", "id", out)
					continue
				*/
			}
			nodes[i] = b.nodes[ni]
			//log.Info("state", "cur", cur, "i", i, "nodes", nodes)
			i++
		}
		//log.Info("state", "cur", cur, "i", i, "nodes", nodes)
		cur++
	}
	nodes[cur] = b.nodes[containsNode(IdEnd, b.nodes)]
	/*
		i := 1
		nodes := make([]Node, len(b.nodes))
		for _, n := range b.nodes {
			if n.id == IdStart {
				nodes[0] = n
			}
			if n.id == IdEnd {
				nodes[len(nodes)-1] = n
				continue
			}
			for _, out := range n.outgoing {
				if containsNode(out, nodes) >= 0 {
					continue
				}
				nodes[i] = b.nodes[containsNode(out, b.nodes)]
				i++
			}
		}
	*/
	b.nodes = nodes
	return nil
}

type AdapterBuilder struct {
	ns *part
	sb *StoryBuilder
}

func (b *AdapterBuilder) Adapter(a string) *OutgoingBuilder {
	log.Debug("setting adapter", "id", a)
	b.ns.adapter = a
	return &OutgoingBuilder{
		ns: b.ns,
		sb: b.sb,
	}
}

type OutgoingBuilder struct {
	ns *part
	sb *StoryBuilder
}

func (b *OutgoingBuilder) LinkTo(ids ...string) *StoryBuilder {
	log.Debug("adding links to saga", "ids", ids)
	b.ns.Outgoing(ids...)
	return b.sb
}

func (b *OutgoingBuilder) LinkToEnd() *StoryBuilder {
	return b.LinkTo(IdEnd)
}

type Part interface {
	Adapter() string
	Id() string
	Next() []string
	Prev() []string
}

type part struct { //State
	id        string
	incomming []string
	outgoing  []string
	adapter   string
}

func (p *part) Outgoing(ids ...string) {
	for _, id := range ids {
		if contains(id, p.outgoing) >= 0 {
			continue
		}
		p.outgoing = append(p.outgoing, id)
	}
}

func (p *part) Inngoing(ids ...string) {
	for _, id := range ids {
		if contains(id, p.incomming) >= 0 {
			continue
		}
		p.incomming = append(p.incomming, id)
	}
}

func (p part) Id() string {
	return p.id
}

func (p part) Adapter() string {
	return p.adapter
}

func (p part) Next() []string {
	return p.outgoing
}

func (p part) Prev() []string {
	return p.incomming
}

//func (s *Saga) Id(id string

func contains(id string, ids []string) int {
	for i, v := range ids {
		if v == id {
			return i
		}
	}
	return -1
}

func containsNode(id string, nodes []part) int {
	for i, n := range nodes {
		if n.id == id {
			return i
		}
	}
	return -1
}

/*
func AdapterLoader(vals ...any) []adapter.Adapter {
	a := make([]adapter.Adapter, len(vals))
	for i, v := range vals {
		a[i] = adapter.New(adapter.Name(v), func(any) (any, error) {
			return nil, nil
		})
	}
	return a
}
*/

var ErrMissingNode = errors.New("Missing node in graph")
var ErrSyclicGraph = errors.New("Graph is syclic")
