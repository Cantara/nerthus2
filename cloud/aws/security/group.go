package security

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	log "github.com/cantara/bragi/sbragi"

	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type Group struct {
	Cluster string `json:"-"`
	Name    string `json:"name"`
	Desc    string `json:"-"`
	Id      string `json:"id"`
	vpc     string
}

/*
func NewDB(serviceName, scope string, vpc vpc.VPC, e2 *ec2.Client) (g Group, err error) {
	g = Group{
		Scope: scope,
		Name:  fmt.Sprintf("%s-%s-db", g.Scope, serviceName),
		Desc:  "Database security group for scope: " + g.Scope + " " + serviceName,
		vpc:   vpc,
		ec2:   e2,
	}
	return
}
*/

func Get(name, vpcId string, e2 *ec2.Client) (g Group, err error) {
	name = strings.ToLower(name)
	log.Trace("getting security group", "name", name)
	result, err := e2.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2types.Filter{
			{
				Name: aws.String("group-name"),
				Values: []string{
					name,
				},
			},
			{
				Name: aws.String("vpc-id"),
				Values: []string{
					vpcId,
				},
			},
		},
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe Security groups, err: %v", err)
		return
	}
	if len(result.SecurityGroups) == 0 {
		log.Trace("no security group found")
		err = ErrNoSecurityGroupsFound
		return
	}

	sg := result.SecurityGroups[0]
	g = Group{
		Cluster: vpc.Tag(sg.Tags, "Cluster"),
		Name:    name,
		Desc:    *sg.Description,
		Id:      *sg.GroupId,
		vpc:     *sg.VpcId,
	}

	return
}
func ById(id string, e2 *ec2.Client) (g Group, err error) {
	log.Trace("getting security group", "id", id)
	result, err := e2.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{id}, //Documentation is weird, might need to use filter instead.
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe Security groups, err: %v", err)
		return
	}
	if len(result.SecurityGroups) == 0 {
		log.Trace("no security group found")
		err = ErrNoSecurityGroupsFound
		return
	}

	sg := result.SecurityGroups[0]
	g = Group{
		Cluster: vpc.Tag(sg.Tags, "Cluster"),
		Name:    *sg.GroupName,
		Desc:    *sg.Description,
		Id:      *sg.GroupId,
		vpc:     *sg.VpcId,
	}

	return
}

func New(name, cluster string, vpc string, e2 *ec2.Client) (Group, error) {
	g, err := Get(name, vpc, e2)
	if err != nil && !errors.Is(err, ErrNoSecurityGroupsFound) {
		return Group{}, err
	}
	if g.Id != "" {
		return g, nil
	}
	g = Group{
		Cluster: cluster,
		Name:    name,
		Desc:    "Security group for cluster: " + cluster,
		vpc:     vpc,
	}
	secGroupRes, err := e2.CreateSecurityGroup(context.Background(), &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(g.Name),
		Description: aws.String(g.Desc),
		VpcId:       aws.String(g.vpc),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeSecurityGroup,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(g.Name),
					},
					{
						Key:   aws.String("Cluster"),
						Value: aws.String(g.Cluster),
					},
				},
			},
		},
	})
	if err != nil {
		return Group{}, err
	}
	g.Id = aws.ToString(secGroupRes.GroupId)

	return g, nil
}

func Wait(id string, e2 *ec2.Client) (err error) {
	err = ec2.NewSecurityGroupExistsWaiter(e2).Wait(context.Background(), &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{
			id,
		},
	}, 5*time.Minute)
	return
}

/*
func (g *Group) Delete() (err error) {
	if !g.created {
		return
	}
	err = util.CheckEC2Session(g.ec2)
	if err != nil {
		return
	}
	_, err = g.ec2.DeleteSecurityGroup(context.Background(), &ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(g.Id),
	})
	return
}
*/

func (g Group) OpenSSH(user string, ip string, e2 *ec2.Client) (err error) {
	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(g.Id),
		IpPermissions: []ec2types.IpPermission{
			{
				FromPort:   aws.Int32(22),
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int32(22),
				IpRanges: []ec2types.IpRange{
					{
						CidrIp:      aws.String(fmt.Sprintf("%s/32", ip)),
						Description: aws.String(fmt.Sprintf("SSH access for %s from %s opened %s", user, ip, time.Now().Format(time.RFC3339))),
					},
				},
			},
		},
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeSecurityGroupRule,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(fmt.Sprintf("SSH access for %s", user)),
					},
					{
						Key:   aws.String("Cluster"),
						Value: aws.String(g.Cluster),
					},
				},
			},
		},
	}

	_, err = e2.AuthorizeSecurityGroupIngress(context.Background(), input)
	if err != nil {
		err = fmt.Errorf("Could not add base authorization to security group %s %s. err: %v", g.Id, g.Name, err)
		return
	}

	return
}

/*
func (g Group) AddDatabaseAuthorization(serverSgId string) (err error) {
	err = util.CheckEC2Session(g.ec2)
	if err != nil {
		return
	}
	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(g.Id),
		IpPermissions: []ec2types.IpPermission{
			{
				FromPort:   aws.Int32(5432),
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int32(5432),
				UserIdGroupPairs: []ec2types.UserIdGroupPair{
					{
						Description: aws.String("Postgresql access from server"),
						GroupId:     aws.String(serverSgId),
					},
				},
			},
		},
	}

	_, err = g.ec2.AuthorizeSecurityGroupIngress(context.Background(), input)
	if err != nil {
		err = util.CreateError{
			Text: fmt.Sprintf("Could not add base authorization to security group %s %s.", g.Id, g.Name),
			Err:  err,
		}
		return
	}

	return
}
*/

func (g Group) AddLoadbalancerAuthorization(loadbalancerId string, port int, e2 *ec2.Client) (err error) {
	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(g.Id),
		IpPermissions: []ec2types.IpPermission{
			{
				FromPort:   aws.Int32(int32(port)),
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int32(int32(port)),
				UserIdGroupPairs: []ec2types.UserIdGroupPair{
					{
						Description: aws.String("HTTP access from loadbalancer"),
						GroupId:     aws.String(loadbalancerId),
					},
				},
			},
		},
	}

	_, err = e2.AuthorizeSecurityGroupIngress(context.Background(), input)
	if err != nil {
		err = fmt.Errorf("Could not add service loadbalancer authorization to security group %s %s. err: %v", g.Id, g.Name, err)
		return
	}
	return
}

func (g Group) AddLoadbalancerPublicAccess(e2 *ec2.Client) (err error) {
	rules, err := e2.DescribeSecurityGroupRules(context.Background(), &ec2.DescribeSecurityGroupRulesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("group-id"),
				Values: []string{g.Id},
			},
		},
	})
	if err != nil {
		err = fmt.Errorf("Could not get service loadbalancer authorization rules for security group %s %s. err: %v", g.Id, g.Name, err)
		return
	}
	V4HTTP := true
	V4HTTPS := true
	V6HTTP := true
	V6HTTPS := true
	for _, rule := range rules.SecurityGroupRules {
		if aws.ToBool(rule.IsEgress) {
			log.Trace("ignoring egress rules for now")
			continue
		}
		if aws.ToString(rule.IpProtocol) != "tcp" {
			err = fmt.Errorf("protocol was not tcp: %s", aws.ToString(rule.IpProtocol))
			return
			/*
				log.Trace("unknown / unrelated protocol", "group", g.Id, "rule", rule.SecurityGroupRuleId)
				continue
			*/
		}
		if rule.FromPort == nil || rule.ToPort == nil {
			err = fmt.Errorf("malconfigured tcp rule. Either from or to port was nil")
			return
		}
		if *rule.FromPort != *rule.ToPort {
			err = fmt.Errorf("to port was not same as from port, %d!=%d", *rule.ToPort, *rule.FromPort)
			return
		}
		switch aws.ToInt32(rule.FromPort) {
		case 80:
			if aws.ToString(rule.CidrIpv4) == "0.0.0.0/0" {
				V4HTTP = false
			}
			if aws.ToString(rule.CidrIpv6) == "::/0" {
				V6HTTP = false
			}
		case 443:
			if aws.ToString(rule.CidrIpv4) == "0.0.0.0/0" {
				V4HTTPS = false
			}
			if aws.ToString(rule.CidrIpv6) == "::/0" {
				V6HTTPS = false
			}
		default:
			err = fmt.Errorf("unsuported / unexpected port: %d", aws.ToInt32(rule.FromPort))
			return
		}
	}
	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(g.Id),
	}
	if V4HTTP || V6HTTP {
		ipPerm := ec2types.IpPermission{
			FromPort:   aws.Int32(80),
			IpProtocol: aws.String("tcp"),
			ToPort:     aws.Int32(80),
		}
		if V4HTTP {
			ipPerm.IpRanges = []ec2types.IpRange{
				{
					Description: aws.String("HTTP access to loadbalancer from anywhere"),
					CidrIp:      aws.String("0.0.0.0/0"),
				},
			}
		}
		if V6HTTP {
			ipPerm.Ipv6Ranges = []ec2types.Ipv6Range{
				{
					Description: aws.String("HTTP access to loadbalancer from anywhere"),
					CidrIpv6:    aws.String("::/0"),
				},
			}
		}
		input.IpPermissions = append(input.IpPermissions, ipPerm)
	}
	if V4HTTPS || V6HTTPS {
		ipPerm := ec2types.IpPermission{
			FromPort:   aws.Int32(443),
			IpProtocol: aws.String("tcp"),
			ToPort:     aws.Int32(443),
		}
		if V4HTTPS {
			ipPerm.IpRanges = []ec2types.IpRange{
				{
					Description: aws.String("HTTPS access to loadbalancer from anywhere"),
					CidrIp:      aws.String("0.0.0.0/0"),
				},
			}
		}
		if V6HTTPS {
			ipPerm.Ipv6Ranges = []ec2types.Ipv6Range{
				{
					Description: aws.String("HTTPS access to loadbalancer from anywhere"),
					CidrIpv6:    aws.String("::/0"),
				},
			}
		}
		input.IpPermissions = append(input.IpPermissions, ipPerm)
	}

	if len(input.IpPermissions) == 0 {
		log.Trace("no permissions to set")
		return
	}

	_, err = e2.AuthorizeSecurityGroupIngress(context.Background(), input)
	if err != nil {
		err = fmt.Errorf("Could not add service loadbalancer authorization to security group %s %s. err: %v", g.Id, g.Name, err)
		return
	}
	return
}

/*
func (g Group) AddServerAccess(sgId string) (err error) {
	err = util.CheckEC2Session(g.ec2)
	if err != nil {
		return
	}
	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(g.Id),
		IpPermissions: []ec2types.IpPermission{
			{
				FromPort:   aws.Int32(5432),
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int32(5432),
				UserIdGroupPairs: []ec2types.UserIdGroupPair{
					{
						Description: aws.String("PSQL access from servers in scope: " + g.Scope),
						GroupId:     aws.String(sgId),
					},
				},
			},
		},
	}

	_, err = g.ec2.AuthorizeSecurityGroupIngress(context.Background(), input)
	if err != nil {
		err = util.CreateError{
			Text: fmt.Sprintf("Could not add PSQL access to security group %s %s.", g.Id, g.Name),
			Err:  err,
		}
		return
	}

	return
}
*/

/*
func (g *Group) AuthorizeHazelcast() (err error) {
	err = util.CheckEC2Session(g.ec2)
	if err != nil {
		return
	}
	input := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(g.Id),
		IpPermissions: []ec2types.IpPermission{
			{
				FromPort:   aws.Int32(5700),
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int32(5799),
				UserIdGroupPairs: []ec2types.UserIdGroupPair{
					{
						Description: aws.String("Hazelcast access"),
						GroupId:     aws.String(g.Id),
					},
				},
			},
		},
	}

	_, err = g.ec2.AuthorizeSecurityGroupIngress(context.Background(), input)
	if err != nil {
		err = util.CreateError{
			Text: fmt.Sprintf("Could not add Hazelcast authorization to security group %s %s.", g.Id, g.Name),
			Err:  err,
		}
		return
	}

	return
}

func (g Group) WithEC2(e *ec2.Client) Group {
	g.ec2 = e
	return g
}
*/

var ErrNoSecurityGroupsFound = fmt.Errorf("no security groups found")
