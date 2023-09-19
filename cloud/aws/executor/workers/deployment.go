package workers

import (
	"fmt"
	"strings"

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
	"github.com/cantara/nerthus2/system"
)

type Executor interface {
	Add(executor.Func)
}

func Deploy(sys system.System, env, nerthus, visuale string, e Executor, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *acm.Client) { //TODO: Change fingerprint to take inn config object
	reqList := []listener.Requireing{}
	reqKey := []key.Requireing{}
	reqSN := []sn.Requireing{}
	reqVPC := []vpc.Requireing{}
	for _, cluster := range sys.Clusters {
		reqNode := []node.Requireing{}
		for _, service := range cluster.Services {
			te := target.Executor(elb)
			reqNode = append(reqNode, te)
			re := rule.Executor(elb)
			reqList = append(reqList, re)

			//Move the path creation to config parsing
			tge := tg.Executor(env, sys.Name, cluster.Name, fmt.Sprintf("/%s", strings.ToLower(service.ServiceInfo.Artifact.Id)), *service.WebserverPort, []tg.Requireing{
				te,
				re,
			}, elb)
			reqVPC = append(reqVPC, tge)

		}
		ne := node.Executor(cluster.NodeNames, cluster.Name, sys.Name, env, cluster.InstanceType, nerthus, visuale, reqNode, e2)
		reqKey = append(reqKey, ne)
		reqSN = append(reqSN, ne)
		e.Add(image.Executor(cluster.OSName, cluster.Arch, []image.Requireing{
			ne,
		}, e2).Execute)
		sge := sg.Executor(env, sys.Name, cluster.Name, []sg.Requireing{
			ne,
		}, e2)
		reqVPC = append(reqVPC, sge)
	}

	le := listener.Executor(reqList, elb)

	e.Add(cert.Executor(sys.Domain, env, []cert.Requireing{
		le,
	}, cc, rc).Execute)

	lbe := lb.Executor(env, sys.Name, []lb.Requireing{
		le,
	}, elb)
	reqSN = append(reqSN, lbe)

	e.Add(key.Executor(env, sys.Name, reqKey, e2).Execute)

	lbsge := lbsg.Executor(env, sys.Name, []lbsg.Requireing{
		lbe,
	}, e2)
	reqVPC = append(reqVPC, lbsge)
	sne := sn.Executor(reqSN, e2)
	reqVPC = append(reqVPC, sne)

	e.Add(vpc.Executor(env, sys.Name, sys.CIDR, reqVPC, e2).Execute)
}

func DeployInfra(nodes []string, arch ami.Arch, imageName, network, cluster, system, env, size, nerthus, visuale, domain string, e Executor, e2 *ec2.Client, elb *elbv2.Client, rc *route53.Client, cc *acm.Client) { //TODO: Change fingerprint to take inn config object
	te := target.Executor(elb)
	re := rule.Executor(elb)

	le := listener.Executor([]listener.Requireing{
		re,
	}, elb)

	e.Add(cert.Executor(domain, env, []cert.Requireing{
		le,
	}, cc, rc).Execute)

	lbe := lb.Executor(env, system, []lb.Requireing{
		le,
	}, elb)

	ne := node.Executor(nodes, cluster, system, env, size, nerthus, visuale, []node.Requireing{
		te,
	}, e2)
	e.Add(key.Executor(env, system, []key.Requireing{
		ne,
	}, e2).Execute)
	e.Add(image.Executor(imageName, arch, []image.Requireing{
		ne,
	}, e2).Execute)

	lbsge := lbsg.Executor(env, system, []lbsg.Requireing{
		lbe,
	}, e2)
	sne := sn.Executor([]sn.Requireing{
		ne,
		lbe,
	}, e2)
	sge := sg.Executor(env, system, cluster, []sg.Requireing{
		ne,
	}, e2)
	/*
		tge := tg.Executor(env, cluster, path, port, []tg.Requireing{
			te,
			re,
		}, elb)
	*/

	e.Add(vpc.Executor(env, system, network, []vpc.Requireing{
		lbsge,
		sne,
		sge,
		//tge,
	}, e2).Execute)
}
