package node

import (
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/image"
	keye "github.com/cantara/nerthus2/cloud/aws/executor/workers/key"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sn"
	"github.com/cantara/nerthus2/cloud/aws/server"
)

var Fingerprint = adapter.New[[]server.Server]("CreateOrGetNodes")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) ([]server.Server, error) {
		env := start.Fingerprint.Value(a[0])
		subnets := sn.Fingerprint.Value(a[1])
		img := image.Fingerprint.Value(a[2])
		k := keye.Fingerprint.Value(a[3])
		sg := sg.Fingerprint.Value(a[4])
		servs := make([]server.Server, env.System.Cluster.Size)
		var ids []string
		var extra string
		if env.MachineName != env.System.MachineName {
			extra = fmt.Sprintf("-%s", env.System.MachineName)
		}
		if env.System.MachineName != env.System.Cluster.MachineName {
			extra = fmt.Sprintf("%s-%s", extra, env.System.Cluster.MachineName)
		}
		for i := range servs {
			s, err := server.Create(i, fmt.Sprintf("%s%s-%d", env.MachineName, extra, i+1), env.System.Cluster.Name, env.System.Name, env.Name, env.System.Cluster.Node.Size,
				subnets[i%len(subnets)], env.NerthusURL, env.VisualeURL, env.System.Cluster.DiskSize(), img, k, sg, c)
			log.WithError(err).Trace("while creating nodes", "env", env.Name, "system", env.System.Name, "cluster", env.System.Cluster.Name, "image", img.HName, "subnets", subnets, "node", i)
			if err != nil {
				return nil, err
			}
			servs[i] = s
		}
		server.WaitUntilRunning(ids, c)
		return servs, nil
	}, start.Fingerprint, sn.Fingerprint, image.Fingerprint, keye.Fingerprint, sg.Fingerprint)
}
