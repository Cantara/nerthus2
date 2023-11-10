package workers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/gober/consensus"
	"github.com/cantara/gober/discovery/local"
	"github.com/cantara/gober/stream/event/store/ondisk"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/executor"
	"github.com/cantara/nerthus2/system"
	"github.com/cantara/nerthus2/system/service"
	"github.com/gofrs/uuid"
)

var STREAM_NAME = "TestDeployment_" + uuid.Must(uuid.NewV7()).String()
var testCryptKey = "aPSIX6K3yw6cAWDQHGPjmhuOswuRibjyLLnd91ojdK0="

func TestProvisionSystem(t *testing.T) {
	return
	//dl, _ := log.NewDebugLogger()
	//dl.SetDefault()

	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.WithError(err).Fatal("while getting aws config")
		t.Fatal(err)
	}

	e := executor.NewExecutor()
	var wg sync.WaitGroup
	numRunners := 10
	wg.Add(numRunners)
	for i := 0; i < numRunners; i++ {
		go func() {
			defer wg.Done()
			e.Run()
		}()
	} // imageName, serviceType, path, network, cluster, system, env, size, nerthus, visuale
	names := make([]string, 3)
	for i := range names {
		names[i] = fmt.Sprintf("%s-%s-%d", "test", "nerthus", i+1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := ondisk.Init(STREAM_NAME, ctx)
	if err != nil {
		t.Fatal(err)
	}
	token := "someTestToken"
	p, err := consensus.Init(3134, token, local.New())
	if err != nil {
		t.Fatal(err)
	}
	d, err := New(store, p.AddTopic, testCryptKey, ec2.NewFromConfig(cfg), elbv2.NewFromConfig(cfg), route53.NewFromConfig(cfg), acm.NewFromConfig(cfg), ctx)
	if err != nil {
		t.Fatal(err)
	}
	go p.Run()
	go d.Work()
	/*
		Env:     env,
		System:  sys.Name,
		Cluster: cluster.Name,
		OSName:  sys.OSName,
		Arch:    a,
		Network: sys.CIDR,
		Nodes:   cluster.NodeNames,
		Size:    cluster.InstanceType,
		Nerthus: nerthus,
		Visuale: visuale,
		Path:    cluster.GetWebserverPath(), //FixMe: This needs to be fixed
		Port:    cluster.GetWebserverPort(),
		Base:    cluster.DNSRoot,
	*/

	port := 8989
	d.ProvisionSystem(system.System{
		Name:   "test",
		OSName: "al2023",
		OSArch: ami.ARM64.String(),
		CIDR:   "172.16.0.0/24",
		Clusters: []*system.Cluster{
			{
				Name: "test",
				NodeNames: []string{
					"test-node-1",
				},
				InstanceType: "t4g.nano",
				Services: []*system.Service{
					{
						ServiceInfo: &service.Service{
							APIPath: "p",
						},
						WebserverPort: &port,
					},
				},
				DNSRoot: "test.exoreaction.dev",
			},
		},
	}, "test", "nerthus", "visuale")
	time.Sleep(time.Hour)
	//Deployment(names, 13030, ami.ARM64, "Amazon Linux 2023", "H2A", "/nerthue", "172.31.100.0/24", "nerthus", "nerthus", "test", "t4g.nano", "nerthus.test.exoreaction.dev", "visuale.test.exoreaction.dev", "exoreaction.dev", &e, ec2.NewFromConfig(cfg), elbv2.NewFromConfig(cfg), route53.NewFromConfig(cfg), acm.NewFromConfig(cfg))
	//DeployInfra(names, ami.ARM64, "Amazon Linux 2023", "172.31.100.0/24", "nerthus", "nerthus", "test", "t4g.nano", "nerthus.test.exoreaction.dev", "visuale.test.exoreaction.dev", "exoreaction.dev", &e, ec2.NewFromConfig(cfg), elbv2.NewFromConfig(cfg), route53.NewFromConfig(cfg), acm.NewFromConfig(cfg))
	//wg.Wait()
}
