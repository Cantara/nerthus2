package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/key"
	"github.com/cantara/nerthus2/cloud/aws/security"
	"github.com/cantara/nerthus2/cloud/aws/server"
	"github.com/cantara/nerthus2/cloud/aws/vpc"

	log "github.com/cantara/bragi/sbragi"
)

func main() {
	dl, _ := log.NewDebugLogger()
	dl.SetDefault()

	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.WithError(err).Fatal("while getting aws config")
	}
	ec2client := ec2.NewFromConfig(cfg)

	testCluster := "test-nerthus"
	testVpcName := testCluster + "-vpc"
	//testKeyName := testCluster + "-key"
	//testGroupName := testCluster + "-sg"
	v, err := vpc.NewVPC(testVpcName, "172.31.200.0/24", ec2client)
	if err != nil {
		log.WithError(err).Fatal("while getting vpc")
	}
	v, err = vpc.GetVPC(testVpcName, ec2client)
	if err != nil {
		log.WithError(err).Fatal("while getting vpc")
	}
	vpc.CreateSubnets(v, ec2client)
	ig, err := vpc.NewIG(v, ec2client)
	if err != nil {
		log.WithError(err).Fatal("while getting ig")
	}
	vpc.AddIGtoRT(v.Id, ig, ec2client)
	k, err := key.New(testCluster, ec2client)
	if err != nil {
		log.WithError(err).Fatal("while getting key")
	}
	sg, err := security.New(testCluster, v.Id, ec2client)
	if err != nil {
		log.WithError(err).Error("while creating new security group")
	}
	img, err := ami.GetImage("Amazon Linux 2023", ami.AMD64, ec2client)
	if err != nil {
		log.WithError(err).Error("while getting ami")
	}
	s, err := vpc.GetSubnets(v.Id, ec2client)
	if err != nil {
		log.WithError(err).Error("while getting subnets")
	}
	fmt.Println(sg)
	fmt.Println(sg.OpenSSH("sindre", "106.155.5.188", ec2client))
	nodes := []string{
		testCluster + "-1",
		testCluster + "-2",
		testCluster + "-3",
	}
	//var wg sync.WaitGroup
	//wg.Add(len(nodes))
	for i := range nodes {
		//go func(i int) {
		func(i int) {
			//defer wg.Done()
			fmt.Println(server.Create(i, nodes, 13030, "H2A", testCluster, "nerthus", "test", "t3.small", s[i].Id, "nerthus.text.exoreaction.dev", "visuale.test.exoreaction.dev", img, k, sg, ec2client))
		}(i)
	}
	//wg.Wait()
}
