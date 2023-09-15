package security

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/cloud/aws/vpc"
)

type Rule struct {
	Id       string `json:"id"`
	Desc     string
	Protocol string
	Name     string
	FromPort int
	ToPort   int
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

func (g *Group) Rules(e2 *ec2.Client) (rules []Rule, err error) {
	log.Trace("getting security group rules", "group", g)
	result, err := e2.DescribeSecurityGroupRules(context.Background(), &ec2.DescribeSecurityGroupRulesInput{
		Filters: []ec2types.Filter{
			{
				Name: aws.String("group-id"),
				Values: []string{
					g.Id,
				},
			},
		},
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe Security groups rules, err: %v", err)
		return
	}
	if len(result.SecurityGroupRules) == 0 {
		log.Trace("no security group rules found")
		err = ErrNoSecurityGroupRulesFound
		return
	}

	//rules = make([]Rule, len(result.SecurityGroupRules))
	for _, r := range result.SecurityGroupRules {
		if *r.IsEgress {
			continue
		}
		log.Trace("got security group rule", "group", g.Name, "rule", r)
		rules = append(rules, Rule{
			Id:       *r.SecurityGroupRuleId,
			Desc:     aws.ToString(r.Description),
			Protocol: aws.ToString(r.IpProtocol),
			Name:     vpc.Tag(r.Tags, "Name"),
			FromPort: int(*r.FromPort),
			ToPort:   int(*r.ToPort),
		})
	}

	return
}

func (g *Group) Revoke(id string, e2 *ec2.Client) (err error) {
	log.Trace("getting security group rules", "group", g)
	_, err = e2.RevokeSecurityGroupIngress(context.Background(), &ec2.RevokeSecurityGroupIngressInput{
		GroupId: &g.Id,
		SecurityGroupRuleIds: []string{
			id,
		},
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe Security groups rules, err: %v", err)
		return
	}
	log.Info("revoced rule")
	return
}

var ErrNoSecurityGroupRulesFound = fmt.Errorf("no security group rules found")
