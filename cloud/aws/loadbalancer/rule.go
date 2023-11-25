package loadbalancer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	//"github.com/aws/aws-sdk-go-v2/aws/awserr"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

type Rule struct {
	ARN         string
	listener    Listener
	targetGroup TargetGroup
}

func GetRules(listenerARN string, elb *elbv2.Client) (r []Rule, err error) {
	input := &elbv2.DescribeRulesInput{
		ListenerArn: aws.String(listenerARN),
	}

	result, err := elb.DescribeRules(context.Background(), input)
	if err != nil {
		return
	}

	paths := []string{}
	for _, rule := range result.Rules {
		for _, condition := range rule.Conditions {
			if aws.ToString(condition.Field) != "path-pattern" {
				continue
			}
			for _, path := range condition.PathPatternConfig.Values {
				paths = append(paths, path)
			}
		}
	}
	return
}

func CreateRulePath(l Listener, tg TargetGroup, elb *elbv2.Client) (r Rule, err error) { //TODO: Need to extend this to support host and default routes
	highestPriority, err := l.GetHighestPriority(elb)
	if err != nil {
		return
	}
	path := fmt.Sprintf("/%s", r.targetGroup.UriPath)

	result, err := elb.CreateRule(context.Background(), &elbv2.CreateRuleInput{
		Actions: []elbv2types.Action{
			{
				TargetGroupArn: aws.String(tg.ARN),
				Type:           "forward",
			},
		},
		Conditions: []elbv2types.RuleCondition{
			{
				Field: aws.String("path-pattern"), //This need to support more than this.
				Values: []string{
					path,
					path + "/*",
				},
			},
		},
		ListenerArn: aws.String(string(l)),
		Priority:    aws.Int32(int32(highestPriority + 1)),
	})
	if err != nil {
		return
	}
	r.ARN = aws.ToString(result.Rules[0].RuleArn)
	return
}

func CreateRuleHost(l Listener, tg TargetGroup, hostHeader string, elb *elbv2.Client) (r Rule, err error) { //TODO: Need to extend this to support host and default routes
	highestPriority, err := l.GetHighestPriority(elb)
	if err != nil {
		return
	}

	result, err := elb.CreateRule(context.Background(), &elbv2.CreateRuleInput{
		Actions: []elbv2types.Action{
			{
				TargetGroupArn: aws.String(tg.ARN),
				Type:           "forward",
			},
		},
		Conditions: []elbv2types.RuleCondition{
			{
				Field: aws.String("host-header"),
				Values: []string{
					hostHeader,
				},
			},
		},
		ListenerArn: aws.String(string(l)),
		Priority:    aws.Int32(int32(highestPriority + 1)),
	})
	if err != nil {
		return
	}
	r.ARN = aws.ToString(result.Rules[0].RuleArn)
	return
}

func (r *Rule) Delete(elb *elbv2.Client) (err error) {
	_, err = elb.DeleteRule(context.Background(), &elbv2.DeleteRuleInput{
		RuleArn: aws.String(r.ARN),
	})
	return
}
