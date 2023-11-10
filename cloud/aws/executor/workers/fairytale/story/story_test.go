package story

import (
	"errors"
	"fmt"
	"log"
	"testing"
)

func TestStory(t *testing.T) {
	s, err := Start("test saga").
		LinkTo("rdbms", "graph", "objectstore").
		Id("graph").Adapter("WriteToGraph").LinkTo("barf").
		Id("barf").Adapter("WriteToBarf").LinkTo("pubsub").
		Id("rdbms").Adapter("WriteToRDBMS").LinkTo("pubsub", "barf").
		Id("pubsub").Adapter("PublishToPubSub").LinkToEnd().
		Id("objectstore").Adapter("WriteToObjectStore").LinkTo("pubsub").
		End()
	if err != nil {
		log.Fatal(err)
	}

	order := []string{IdStart, "rdbms", "graph", "objectstore", "barf", "pubsub", IdEnd}

	if len(order) != len(s.Parts()) {
		t.Fatalf("length of expected order != length of nodes in saga (%d != %d)\n%v\n", len(order), len(s.Parts()), s)
	}
	for i, n := range s.Parts() {
		if order[i] != n.Id() {
			t.Fatalf("wrong object at index(%d), %s != %s\n%v\n", i, order[i], n.Id(), s)
		}
	}
	fmt.Println(s)
}

func TestMissingNode(t *testing.T) {
	_, err := Start("test saga").LinkTo("graph").
		End()
	if !errors.Is(err, ErrMissingNode) {
		t.Fatal("no missing node error", err)
	}

}

func TestValidateSycle(t *testing.T) {
	_, err := Start("test saga").LinkTo("graph").
		Id("graph").Adapter("WriteToGraph").LinkTo("graph2").
		Id("graph2").Adapter("WriteToGraph").LinkTo("graph").
		End()
	if !errors.Is(err, ErrSyclicGraph) {
		t.Fatal("syclic graph reported as asyclic", err)
	}

	_, err = Start("test saga").LinkTo("graph").
		Id("graph").Adapter("WriteToGraph").LinkToEnd().
		End()
	if err != nil {
		t.Fatalf("asyclic graph reported as syclic")
	}
}
