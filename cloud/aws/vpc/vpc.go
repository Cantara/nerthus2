package vpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	log "github.com/cantara/bragi/sbragi"
)

type VPC struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	CIDR   string `json:"cidr"`
	CIDRv6 string `json:"cidr_v6"`
}

func GetVPC(name string, e2 *ec2.Client) (vpc VPC, err error) {
	name = strings.ToLower(name)
	log.Trace("getting vpc", "name", name)
	result, err := e2.DescribeVpcs(context.Background(), &ec2.DescribeVpcsInput{
		/* TODO: explore why this does not work
		Filters: []ec2types.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []string{
					name,
				},
			},
		},
		*/
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe VPCs, err: %v", err)
		return
	}
	if len(result.Vpcs) == 0 {
		log.Trace("no vpcs found")
		err = ErrNoVPCsFound
		return
	}

	for _, v := range result.Vpcs {
		vpcName := Tag(v.Tags, "name")
		log.Trace("checking vpc", "name", name, "found", vpcName)
		if strings.ToLower(vpcName) != name {
			continue
		}
		log.Trace("found vpc", "name", name, "id", *v.VpcId, "cidr", *v.CidrBlock, "name", vpcName)
		vpc = VPC{
			Id:     aws.ToString(v.VpcId),
			Name:   name,
			CIDR:   *v.CidrBlock,
			CIDRv6: *v.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock, //TODO: This can error
		}
		return
	}

	err = ErrNoVPCsFound
	return
}

func ValidateCIDR(cidr string) error {
	if !strings.HasSuffix(cidr, "/24") {
		return fmt.Errorf("cidr block not of size /24, %s", cidr)
	}
	octects := strings.Split(strings.TrimSuffix(cidr, "/24"), ".")
	if len(octects) != 4 {
		return fmt.Errorf("cidr block does not contain 4 octets, %d", len(octects))
	}
	if octects[3] != "0" {
		return fmt.Errorf("cidr last octet if not 0, %s", octects[3])
	}

	return nil
}

func NewVPC(name string, cidr string, e2 *ec2.Client) (vpc VPC, err error) {
	err = ValidateCIDR(cidr)
	if err != nil {
		return
	}
	vpc, err = GetVPC(name, e2)
	if err != nil {
		if !errors.Is(err, ErrNoVPCsFound) {
			return VPC{}, err
		}
	} else {
		return vpc, nil
	}
	log.Trace("creating new vpc", "name", name)

	result, err := e2.CreateVpc(context.Background(), &ec2.CreateVpcInput{
		AmazonProvidedIpv6CidrBlock: aws.Bool(true),
		CidrBlock:                   aws.String(cidr),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeVpc,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(name),
					},
				},
			},
		},
	})

	if err != nil {
		return VPC{}, err
	}

	if result == nil {
		return VPC{}, fmt.Errorf("result was nil when creating vpc %s with cidr %s", name, cidr)
	}

	id := *result.Vpc.VpcId
	ipv6Cidr := *result.Vpc.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock //TODO: This can error
	log.Trace("created vpc", "id", id, "name", name, "cidr", cidr, "ipv6Cidr", ipv6Cidr)

	_, err = e2.ModifyVpcAttribute(context.Background(), &ec2.ModifyVpcAttributeInput{
		VpcId:            &id,
		EnableDnsSupport: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
	})
	_, err = e2.ModifyVpcAttribute(context.Background(), &ec2.ModifyVpcAttributeInput{
		VpcId:              &id,
		EnableDnsHostnames: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
	})

	vpc = VPC{
		Id:     id,
		Name:   name,
		CIDR:   cidr,
		CIDRv6: ipv6Cidr,
	}

	return
}

func Tag(tags []ec2types.Tag, key string) string {
	for _, tag := range tags {
		if strings.ToLower(aws.ToString(tag.Key)) != key {
			continue
		}
		return aws.ToString(tag.Value)
	}
	return ""
}

var ErrNoVPCsFound = fmt.Errorf("No VPCs found.")
