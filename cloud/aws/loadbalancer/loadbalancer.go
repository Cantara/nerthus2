package loadbalancer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/smithy-go"
)

type Loadbalancer struct {
	ARN           string `json:"arn"`
	Name          string
	SecurityGroup string   `json:"security_group"`
	DNSName       string   `json:"dns_name"`
	ListenerARN   string   `json:"listener_arn"`
	Paths         []string `json:"paths"`
}

func GetLoadbalancers(svc *elbv2.Client) (loadbalancers []Loadbalancer, err error) {
	input := &elbv2.DescribeLoadBalancersInput{}

	result, err := svc.DescribeLoadBalancers(context.Background(), input)
	if err != nil {
		var lbnf *elbv2types.LoadBalancerNotFoundException
		var apiErr smithy.APIError
		if errors.As(err, &lbnf) {
			code := lbnf.ErrorCode()
			message := lbnf.ErrorMessage()
			fmt.Printf("%s[%s]:%v\n", "LoadBalancerNotFoundException", code, message)
			return
		} else if errors.As(err, &apiErr) {
			code := apiErr.ErrorCode()
			message := apiErr.ErrorMessage()
			fmt.Printf("%s[%s]:%v\n", "Default", code, message)
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	for _, loadbalancer := range result.LoadBalancers {
		if loadbalancer.Type != "application" {
			continue
		}
		if loadbalancer.Scheme != "internet-facing" {
			//We could continue here if we don't want internal loadbalancers
			continue
		}
		result2, err := svc.DescribeListeners(context.Background(), &elbv2.DescribeListenersInput{
			LoadBalancerArn: loadbalancer.LoadBalancerArn,
		})
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) {
				code := apiErr.ErrorCode()
				message := apiErr.ErrorMessage()
				fmt.Printf("%s[%s]:%v\n", "Default", code, message)
			} else {
				fmt.Println(err.Error())
			}
			return nil, err
		}
		for _, listener := range result2.Listeners {
			if *listener.Port != 443 {
				continue
			}
			if listener.Protocol != "HTTPS" {
				fmt.Println(listener.Protocol)
				continue
			}
			input3 := &elbv2.DescribeRulesInput{
				ListenerArn: listener.ListenerArn,
			}

			result3, err := svc.DescribeRules(context.Background(), input3)
			if err != nil {
				var apiErr smithy.APIError
				if errors.As(err, &apiErr) {
					code := apiErr.ErrorCode()
					message := apiErr.ErrorMessage()
					fmt.Printf("%s[%s]:%v\n", "Default", code, message)
				} else {
					fmt.Println(err.Error())
				}
				return nil, err
			}
			var paths []string
			for _, rule := range result3.Rules {
				for _, condition := range rule.Conditions {
					if *condition.Field != "path-pattern" {
						continue
					}
					for _, path := range condition.PathPatternConfig.Values {
						paths = append(paths, path)
					}
				}
			}
			loadbalancers = append(loadbalancers, Loadbalancer{
				ARN:           aws.ToString(loadbalancer.LoadBalancerArn),
				Name:          aws.ToString(loadbalancer.LoadBalancerName),
				SecurityGroup: loadbalancer.SecurityGroups[0],
				DNSName:       aws.ToString(loadbalancer.DNSName),
				ListenerARN:   aws.ToString(listener.ListenerArn),
				Paths:         paths,
			})
			break
		}
	}
	return
}

func CreateLoadbalancer(name, sg string, subnets []string, elb *elbv2.Client) (Loadbalancer, error) {
	err := ValidateString(name, ValidChars, 3, 32)
	if err != nil {
		return Loadbalancer{}, err
	}
	result, err := elb.CreateLoadBalancer(context.TODO(), &elbv2.CreateLoadBalancerInput{
		Name:           &name,
		IpAddressType:  elbv2types.IpAddressTypeDualstack,
		Scheme:         elbv2types.LoadBalancerSchemeEnumInternetFacing,
		SecurityGroups: []string{sg},
		Subnets:        subnets,
	})

	if err != nil {
		return Loadbalancer{}, err
	}
	if len(result.LoadBalancers) != 1 {
		return Loadbalancer{}, fmt.Errorf("did not create loadbalancer but also no error, number of loadbalancers returned %d", len(result.LoadBalancers))
	}

	lb := result.LoadBalancers[0]
	return Loadbalancer{
		ARN:     aws.ToString(lb.LoadBalancerArn),
		Name:    name,
		DNSName: aws.ToString(lb.DNSName),
	}, nil
}

var ValidChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890-"

// This might be a way to detailed way of handling validation
func ValidateString(s, alphabet string, min, max int) error {
	errs := strings.Builder{}
	if len(s) > max {
		errs.WriteString("string too long")
	} else if len(s) < min {
		errs.WriteString("string too short")
	}
	invalid := strings.Builder{}
	invalid.WriteRune('[')
	for _, c := range s {
		if strings.ContainsRune(alphabet, c) {
			continue
		}
		invalid.WriteRune(c)
		invalid.WriteRune(',')
	}
	if invalid.Len() > 1 {
		errs.WriteString(fmt.Sprintf("invalid chars: %s]", invalid.String()[:invalid.Len()-1]))
	}
	if errs.Len() > 0 {
		return errors.New(errs.String())
	}
	return nil
}
