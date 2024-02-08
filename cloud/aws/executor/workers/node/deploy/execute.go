package deploy

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/sync"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/node"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/message"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigDefault

var Fingerprint = adapter.New[[]string]("ConfigNodes")

func Adapter(nodeActions sync.Map[chan message.Action], c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (out []string, err error) {
		env := start.Fingerprint.Value(a[0])
		nodes := node.Fingerprint.Value(a[1])
		data, err := json.Marshal(env)
		if err != nil {
			return nil, err
		}
		for _, node := range nodes {
			log.Trace("building config for node", "node", node.Id, "config", env)
			na, _ := nodeActions.GetOrInit(node.Name, func() chan message.Action {
				return make(chan message.Action, 10)
			})
			na <- message.Action{
				Action: message.Config,
				Data:   data,
			}
			out = append(out, node.Id)
		}
		return
	}, start.Fingerprint, node.Fingerprint)
}
