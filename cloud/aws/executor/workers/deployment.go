package workers

import (
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/cert"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/image"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/key"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/lb"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/listener"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/node"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/rule"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/target"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/lbsg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sg"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/sn"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/vpc/tg"
)

type Executor interface {
	Add(executor.Func)
}

func Deployment(nodes []string, port int, arch ami.Arch, imageName, serviceType, path, network, cluster, system, env, size, nerthus, visuale, domain string, e Executor, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *acm.Client) { //TODO: Change fingerprint to take inn config object
	te := target.Executor(elb)
	re := rule.Executor(elb)

	le := listener.Executor([]listener.Requireing{
		re,
	}, elb)

	e.Add(cert.Executor(domain, env, []cert.Requireing{
		le,
	}, cc, rc).Execute)

	lbe := lb.Executor(env, cluster, []lb.Requireing{
		le,
	}, elb)

	ne := node.Executor(nodes, port, serviceType, cluster, system, env, size, nerthus, visuale, []node.Requireing{
		te,
	}, e2)
	e.Add(key.Executor(env, cluster, []key.Requireing{
		ne,
	}, e2).Execute)
	e.Add(image.Executor(imageName, arch, []image.Requireing{
		ne,
	}, e2).Execute)

	lbsge := lbsg.Executor(env, cluster, []lbsg.Requireing{
		lbe,
	}, e2)
	sne := sn.Executor([]sn.Requireing{
		ne,
		lbe,
	}, e2)
	sge := sg.Executor(env, cluster, []sg.Requireing{
		ne,
	}, e2)
	tge := tg.Executor(env, cluster, path, port, []tg.Requireing{
		te,
		re,
	}, elb)

	e.Add(vpc.Executor(env, cluster, network, []vpc.Requireing{
		lbsge,
		sne,
		sge,
		tge,
	}, e2).Execute)
}
