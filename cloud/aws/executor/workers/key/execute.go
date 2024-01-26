package key

import (
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
	"github.com/cantara/nerthus2/cloud/aws/key"
)

var Fingerprint = adapter.New[key.Key]("NewKey")

func Adapter(c *ec2.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (k key.Key, err error) {
		env := start.Fingerprint.Value(a[0])
		var extra string
		if env.MachineName != env.System.MachineName {
			extra = fmt.Sprintf("-%s", env.System.MachineName)
		}
		if env.System.MachineName != env.System.Cluster.MachineName {
			extra = fmt.Sprintf("%s-%s", extra, env.System.Cluster.MachineName)
		}
		k, err = key.New(fmt.Sprintf("%s%s-key", env.MachineName, extra), c)
		log.WithError(err).Trace("creating key")
		return
	}, start.Fingerprint)
}
