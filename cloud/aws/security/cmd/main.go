package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/security"
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
	testGroupName := testCluster + "-sg"
	v, err := vpc.NewVPC(testVpcName, "172.31.200.0/24", ec2client)
	if err != nil {
		log.WithError(err).Fatal("while getting vpc")
	}
	fmt.Println(security.Get(testGroupName, v.Id, ec2client))
	sg, err := security.New(testCluster, v.Id, ec2client)
	if err != nil {
		log.WithError(err).Error("while creating new security group")
	}
	fmt.Println(sg)
	fmt.Println(security.Get(testGroupName, v.Id, ec2client))
	fmt.Println(sg.OpenSSH("sindre", "192.168.12.10", ec2client))
	fmt.Println(security.Get(testGroupName, v.Id, ec2client))
	r, err := sg.Rules(ec2client)
	if err != nil {
		log.WithError(err).Error("while getting rules")
	}
	fmt.Println(r)
	fmt.Println(sg.Revoke(r[0].Id, ec2client))
	fmt.Println(sg.Rules(ec2client))
}
