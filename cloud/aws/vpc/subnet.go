package vpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/cloud/aws/az"
)

type Subnet struct {
	VPC    string
	AZ     string
	Id     string
	Name   string
	CIDR   string
	CIDRv6 string
}

func SubnetsToIds(subnets []Subnet) (out []string) {
	out = make([]string, len(subnets))
	for i, s := range subnets {
		out[i] = s.Id
	}

	return
}

func GetSubnets(v string, e2 *ec2.Client) (subnets []Subnet, err error) {
	v = strings.ToLower(v)
	log.Trace("getting subnets", "vpc", v)
	result, err := e2.DescribeSubnets(context.Background(), &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []string{
					v,
				},
			},
		},
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe Subnets, err: %v", err)
		return
	}
	if len(result.Subnets) == 0 {
		log.Trace("no subnets found")
		err = ErrNoSubnetsFound
		return
	}

	for _, s := range result.Subnets {
		name := Tag(s.Tags, "name")
		subnets = append(subnets, Subnet{
			VPC:    v,
			AZ:     *s.AvailabilityZoneId,
			Id:     *s.SubnetId,
			Name:   name,
			CIDR:   *s.CidrBlock,
			CIDRv6: *s.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock, //TODO: this can error
		})
	}

	return
}

func CreateSubnets(vpc VPC, e2 *ec2.Client) (err error) {
	azs, err := az.GetAZs(e2)
	if err != nil {
		return
	}
	subnets, err := GetSubnets(vpc.Id, e2)
	if err != nil && !errors.Is(err, ErrNoSubnetsFound) {
		return err
	}
	if len(azs) == len(subnets) {
		log.Info("number of subnets equal availability zones")
		return nil
	}
	cidrBase := strings.Join(strings.Split(vpc.CIDR, ".")[:3], ".")
	cidrV6Base := vpc.CIDRv6[:len(vpc.CIDRv6)-6]
	name := strings.TrimSuffix(vpc.Name, "-vpc")
	for i, az := range azs {
		if subnetInAz(az, subnets) {
			continue
		}
		cidr := fmt.Sprintf("%s.%d/26", cidrBase, 64*(i%4))
		cidrV6 := fmt.Sprintf("%s%d::/64", cidrV6Base, i)
		result, err := e2.CreateSubnet(context.Background(), &ec2.CreateSubnetInput{
			VpcId:              &vpc.Id,
			AvailabilityZoneId: &az,
			CidrBlock:          &cidr,
			Ipv6CidrBlock:      &cidrV6,
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeSubnet,
					Tags: []ec2types.Tag{
						{
							Key:   aws.String("Name"),
							Value: aws.String(fmt.Sprintf("%s-subnet-%d", name, i+1)),
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		log.Trace("created subnet", "result", result)
	}
	return
}

func subnetInAz(az string, subnets []Subnet) bool {
	for _, s := range subnets {
		if s.AZ == az {
			return true
		}
	}
	return false
}

var ErrNoSubnetsFound = fmt.Errorf("no subnets found")
