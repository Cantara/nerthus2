package workers

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/cloud/aws/executor"
)

func TestDeployment(t *testing.T) {
	dl, _ := log.NewDebugLogger()
	dl.SetDefault()

	// Load the Shared AWS Configuration (~/.aws/config)
	_, err := config.LoadDefaultConfig(context.TODO())
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
	//Deployment(names, 13030, ami.ARM64, "Amazon Linux 2023", "H2A", "/nerthue", "172.31.100.0/24", "nerthus", "nerthus", "test", "t4g.nano", "nerthus.test.exoreaction.dev", "visuale.test.exoreaction.dev", "exoreaction.dev", &e, ec2.NewFromConfig(cfg), elbv2.NewFromConfig(cfg), route53.NewFromConfig(cfg), acm.NewFromConfig(cfg))
	//DeployInfra(names, ami.ARM64, "Amazon Linux 2023", "172.31.100.0/24", "nerthus", "nerthus", "test", "t4g.nano", "nerthus.test.exoreaction.dev", "visuale.test.exoreaction.dev", "exoreaction.dev", &e, ec2.NewFromConfig(cfg), elbv2.NewFromConfig(cfg), route53.NewFromConfig(cfg), acm.NewFromConfig(cfg))
	//wg.Wait()
}
