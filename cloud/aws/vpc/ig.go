package vpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	log "github.com/cantara/bragi/sbragi"
)

func VPCHasIG(v string, e2 *ec2.Client) (id string, err error) {
	v = strings.ToLower(v)
	log.Trace("getting internet gateway", "vpc", v)
	result, err := e2.DescribeInternetGateways(context.Background(), &ec2.DescribeInternetGatewaysInput{
		Filters: []ec2types.Filter{
			{
				Name: aws.String("attachment.vpc-id"),
				Values: []string{
					v,
				},
			},
		},
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe internet gateway, err: %v", err)
		return
	}
	if len(result.InternetGateways) == 0 {
		log.Trace("no internet gateway found")
		return "", nil
	}

	if len(result.InternetGateways) > 1 {
		log.Warning("more than one internet gateway found", "vpc", v, "igs", result.InternetGateways)
	}

	return *result.InternetGateways[0].InternetGatewayId, nil
}

func NewIG(vpc VPC, e2 *ec2.Client) (id string, err error) {
	id, err = VPCHasIG(vpc.Id, e2)
	if err != nil {
		return "", err
	}
	if id != "" {
		log.Trace("vpc has internet gateway", "vpc", vpc.Name, "ig", id)
		return id, nil
	}
	name := strings.TrimSuffix(vpc.Name, "-vpc")
	result, err := e2.CreateInternetGateway(context.Background(), &ec2.CreateInternetGatewayInput{
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInternetGateway,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(fmt.Sprintf("%s-ig", name)),
					},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}
	_, err = e2.AttachInternetGateway(context.Background(), &ec2.AttachInternetGatewayInput{
		InternetGatewayId: result.InternetGateway.InternetGatewayId,
		VpcId:             &vpc.Id,
	})
	if err != nil {
		return "", err
	}
	log.Trace("created internet gateway", "result", result)
	id = *result.InternetGateway.InternetGatewayId
	return
}
