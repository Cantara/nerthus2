package cleanup

import (
	"errors"

	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/node"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/server"
)

var Fingerprint = adapter.New[[]string]("CleanupNodes")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (terminated []string, err error) {
		env := start.Fingerprint.Value(a[0])
		nodes := node.Fingerprint.Value(a[1])
		for _, node := range nodes {
			servers, err := server.GetServers(node.Name, c)
			if err != nil {
				if !errors.Is(err, server.ErrServerNotFound) {
					return nil, err
				}
				continue
			}
			terminate := false
			for _, s := range servers {
				if s.Type == env.System.Cluster.Node.Size && s.State == string(ec2types.InstanceStateNameRunning) {
					terminate = true
					break
				}
			}
			if !terminate {
				continue
			}
			for _, s := range servers {
				if s.Type == env.System.Cluster.Node.Size || s.State != string(ec2types.InstanceStateNameRunning) {
					continue
				}
				err = s.Delete(c)
				if log.WithError(err).Trace("deleting server with old configuration") {
					return nil, err
				}
				terminated = append(terminated, s.Id)
			}
		}
		return
	}, start.Fingerprint, node.Fingerprint)
}
