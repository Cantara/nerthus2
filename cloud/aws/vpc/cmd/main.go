package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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

	v, err := vpc.GetVPC("test-nerthus-vpc", ec2client)
	fmt.Println(v, err)
	fmt.Println(vpc.GetRT(v.Id, ec2client))
	ig, err := vpc.NewIG(v, ec2client)
	err = vpc.AddIGtoRT(v.Id, ig, ec2client)
	return

	fmt.Println(cfg.Region)
	fmt.Println(vpc.GetVPC("sf-visuale-vpc", ec2client))
	fmt.Println(vpc.GetVPC("test-nerthus-vpc", ec2client))
	fmt.Println(vpc.NewVPC("test-nerthus-vpc", "172.31.200.0/24", ec2client))
	v, err = vpc.GetVPC("test-nerthus-vpc", ec2client)
	fmt.Println(v, err)
	fmt.Println(vpc.GetSubnets(v.Id, ec2client))
	fmt.Println(vpc.CreateSubnets(v, ec2client))
	fmt.Println(vpc.GetSubnets(v.Id, ec2client))
	fmt.Println(vpc.VPCHasIG(v.Id, ec2client))
	fmt.Println(ig, err)
	fmt.Println(vpc.VPCHasIG(v.Id, ec2client))
	fmt.Println(vpc.GetVPC("test-nerthus-vpc", ec2client))
}
