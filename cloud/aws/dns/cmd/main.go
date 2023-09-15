package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	aacm "github.com/aws/aws-sdk-go-v2/service/acm"

	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/cloud/aws/acm"
)

func main() {
	dl, _ := log.NewDebugLogger()
	dl.SetDefault()

	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.WithError(err).Fatal("while getting aws config")
	}
	acmclient := aacm.NewFromConfig(cfg)

	fmt.Println(acm.GetCert("*.dev.quadim.dev", acmclient))
	c, err := acm.NewCert("*.dev.quadim.dev", acmclient)
	fmt.Println(c, err)
	fmt.Println(acm.GetDomainValidation(c.Id, acmclient))
	fmt.Println(acm.GetCert("*.dev.quadim.dev", acmclient))
}
