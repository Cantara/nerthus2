package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	aacm "github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/cantara/nerthus2/cloud/aws/acm"
	"github.com/cantara/nerthus2/cloud/aws/ami"
	"github.com/cantara/nerthus2/cloud/aws/dns"
	"github.com/cantara/nerthus2/cloud/aws/key"
	"github.com/cantara/nerthus2/cloud/aws/loadbalancer"
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
	fmt.Println(sg.OpenSSH("sindre", "192.168.12.10", ec2client))
	serv, err := server.Create(testCluster+"-1", testCluster, "t3.small", s[0].Id, img, k, sg, ec2client)
	fmt.Println(serv, err)

	elb := elbv2.NewFromConfig(cfg)
	lbName := fmt.Sprintf("%s-lb", testCluster)
	lbsg, err := security.New(lbName, v.Id, ec2client)

	subnets, err := vpc.GetSubnets(v.Id, ec2client)
	lb, err := loadbalancer.CreateLoadbalancer(lbName, lbsg.Id, vpc.SubnetsToIds(subnets), elb)
	fmt.Println(lb, err)
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
	tg, err := loadbalancer.CreateTargetGroup(v.Id, fmt.Sprintf("%s-tg", testCluster), "/nerthus", 3030, elb)
	listner, err := loadbalancer.CreateListener(lb.ARN, c.Id, elb)
	fmt.Println("list", listner, err)
	fmt.Println("tg", tg, err)
	fmt.Println(loadbalancer.CreateTarget(tg, serv.Id, elb))
	fmt.Println(loadbalancer.CreateRule(listner, tg, elb))
}
