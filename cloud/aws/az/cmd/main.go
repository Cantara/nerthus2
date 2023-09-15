package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/az"

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

	fmt.Println(cfg.Region)
	fmt.Println(az.GetAZs(ec2client))
	fmt.Println(az.GetAZs(ec2client))
	fmt.Println(az.GetAZs(ec2client))
	fmt.Println(az.GetAZs(ec2client))
	fmt.Println(az.GetAZs(ec2client))
}
