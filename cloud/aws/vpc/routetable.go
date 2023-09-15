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

type Route struct {
	Dest    string
	Gateway string
}

type RouteTable struct {
	Id     string
	Vpc    string
	Routes []Route
}

func (rt RouteTable) CoutainsRoute(dest string) bool {
	for _, r := range rt.Routes {
		if r.Dest == dest {
			return true
		}
	}
	return false
}

func GetRT(v string, e2 *ec2.Client) (RouteTable, error) {
	v = strings.ToLower(v)
	log.Trace("getting route tables", "vpc", v)
	result, err := e2.DescribeRouteTables(context.Background(), &ec2.DescribeRouteTablesInput{
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
		err = fmt.Errorf("Unable to describe internet gateway, err: %v", err)
		return RouteTable{}, err
	}
	if len(result.RouteTables) == 0 {
		log.Trace("no route table found")
		return RouteTable{}, fmt.Errorf("no route table found vpc=%s", v)
	}

	if len(result.RouteTables) > 1 {
		log.Warning("more than one route table found", "vpc", v, "igs", result.RouteTables)
		return RouteTable{}, fmt.Errorf("too many route tables found, vpc=%s, routetables=%d", v, len(result.RouteTables))
	}

	rt := result.RouteTables[0]
	routes := make([]Route, len(rt.Routes))
	log.Trace("got route table", "vpc", *rt.VpcId, "id", *rt.RouteTableId)
	for i, r := range rt.Routes {
		var dest string
		if r.DestinationCidrBlock != nil {
			dest = *r.DestinationCidrBlock
		} else if r.DestinationIpv6CidrBlock != nil {
			dest = *r.DestinationIpv6CidrBlock
		}
		gw := aws.ToString(r.GatewayId)
		log.Trace("route", "dest", dest, "gateway", gw)
		routes[i] = Route{
			Dest:    dest,
			Gateway: gw,
		}
	}

	return RouteTable{
		Vpc:    v,
		Id:     *rt.RouteTableId,
		Routes: routes,
	}, nil
}

func AddIGtoRT(vpc, ig string, e2 *ec2.Client) (err error) {
	rt, err := GetRT(vpc, e2)
	if err != nil {
		return err
	}
	if !rt.CoutainsRoute("0.0.0.0/0") {
		err = AddRouteToRT(rt.Id, "0.0.0.0/0", ig, e2)
		if err != nil {
			return
		}
	}
	if !rt.CoutainsRoute("::/0") {
		err = AddRouteToRT(rt.Id, "::/0", ig, e2)
		if err != nil {
			return
		}
	}
	return
}

func AddRouteToRT(rt, dest, gateway string, e2 *ec2.Client) error {
	input := ec2.CreateRouteInput{
		RouteTableId: &rt,
		GatewayId:    &gateway,
	}
	if strings.Contains(dest, ":") {
		input.DestinationIpv6CidrBlock = &dest
	} else {
		input.DestinationCidrBlock = &dest
	}
	_, err := e2.CreateRoute(context.Background(), &input)

	return err
}

func NewRT(vpc VPC, e2 *ec2.Client) (err error) {
	igId, err := VPCHasIG(vpc.Id, e2)
	if err != nil {
		return err
	}
	if igId != "" {
		log.Trace("vpc has internet gateway", "vpc", vpc.Name)
		return nil
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
		return err
	}
	_, err = e2.AttachInternetGateway(context.Background(), &ec2.AttachInternetGatewayInput{
		InternetGatewayId: result.InternetGateway.InternetGatewayId,
		VpcId:             &vpc.Id,
	})
	if err != nil {
		return err
	}
	log.Trace("created internet gateway", "result", result)
	return
}
