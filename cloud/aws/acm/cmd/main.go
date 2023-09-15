package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	aacm "github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/route53"

	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/cloud/aws/acm"
	"github.com/cantara/nerthus2/cloud/aws/dns"
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
	r53 := route53.NewFromConfig(cfg)

	domainBaseName := "quadim.dev"
	domainName := fmt.Sprintf("dev.%s", domainBaseName)
	certName := fmt.Sprintf("*.%s", domainName)

	fmt.Println(acm.GetCert(certName, acmclient))
	c, err := acm.NewCert(certName, acmclient)
	fmt.Println(c, err)
	key, val, err := acm.GetDomainValidation(c.Id, acmclient)
	fmt.Println(key, val, err)
	zone, err := dns.GetHostedZoneId(domainBaseName, r53)
	fmt.Println(zone, err)
	fmt.Println(dns.NewRecord(zone, key, val, "cname", r53))
	fmt.Println(acm.GetCert(certName, acmclient))
}
