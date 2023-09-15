package loadbalancer

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbTypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	log "github.com/cantara/bragi/sbragi"
)

type Listener string

func GetListener(arn string, elb *elbv2.Client) (err error) { //FIXME
	return
}

func GetLoadbalancer(listenerArn string, elb *elbv2.Client) (loadbalancer string, err error) {
	result, err := elb.DescribeListeners(context.TODO(), &elbv2.DescribeListenersInput{
		ListenerArns: []string{
			listenerArn,
		},
	})
	if err != nil {
		return
	}
	loadbalancer = *result.Listeners[0].LoadBalancerArn
	return
}

func (l Listener) GetLoadbalancer(elb *elbv2.Client) (loadbalancer string, err error) {
	return GetLoadbalancer(string(l), elb)
}

func GetListeners(loadbalancerARN string, elb *elbv2.Client) (l []Listener, err error) {
	result, err := elb.DescribeListeners(context.TODO(), &elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(loadbalancerARN),
	})
	if err != nil {
		return
	}

	for _, listener := range result.Listeners {
		if *listener.Port != 443 {
			continue
		}
		if listener.Protocol != "HTTPS" {
			continue
		}
		l = append(l, Listener(aws.ToString(listener.ListenerArn))) //This seems stupide, there should not be possible to have multiple listeners with the same port and protocol
	}
	return
}

func CreateListener(loadbalancer, cert string, elb *elbv2.Client) (listener Listener, err error) {
	listeners, _ := GetListeners(loadbalancer, elb)
	if len(listeners) == 1 {
		log.Trace("loadbalancer contains listener")
		return listeners[0], nil
	}
	result, err := elb.CreateListener(context.TODO(), &elbv2.CreateListenerInput{
		LoadBalancerArn: aws.String(loadbalancer),
		//AlpnPolicy:      []string{"HTTP1Only"}, //Should become HTTP2Preferred
		Certificates: []elbTypes.Certificate{
			{
				CertificateArn: &cert,
			},
		},
		DefaultActions: []elbTypes.Action{
			{
				Type: elbTypes.ActionTypeEnumFixedResponse,
				FixedResponseConfig: &elbTypes.FixedResponseActionConfig{ //Should be configurable to be able to use Forward aswell
					ContentType: aws.String("application/json"),
					MessageBody: aws.String(`{"status":"404","error":"page not found"}`),
					StatusCode:  aws.String("404"),
				},
			},
		},
		Port:      aws.Int32(443),
		Protocol:  elbTypes.ProtocolEnumHttps,
		SslPolicy: aws.String("ELBSecurityPolicy-2016-08"),
	})
	if err != nil {
		return
	}
	_, err = elb.CreateListener(context.TODO(), &elbv2.CreateListenerInput{
		LoadBalancerArn: aws.String(loadbalancer),
		AlpnPolicy:      []string{"HTTP1Only"}, //Should become HTTP2Preferred
		DefaultActions: []elbTypes.Action{
			{
				Type: elbTypes.ActionTypeEnumRedirect,
				RedirectConfig: &elbTypes.RedirectActionConfig{
					Host:       aws.String("#{host}"),
					Path:       aws.String("/#{path}"),
					Query:      aws.String("#{query}"),
					Port:       aws.String("443"),
					Protocol:   aws.String("HTTPS"),
					StatusCode: elbTypes.RedirectActionStatusCodeEnumHttp301,
				},
			},
		},
		Port:     aws.Int32(80),
		Protocol: elbTypes.ProtocolEnumHttp,
	})
	if err != nil {
		return
	}
	listener = Listener(*result.Listeners[0].ListenerArn)
	return
}

func (l Listener) GetNumRules(elb *elbv2.Client) (numRules int, err error) {
	result, err := elb.DescribeRules(context.TODO(), &elbv2.DescribeRulesInput{
		ListenerArn: aws.String(string(l)),
	})
	if err != nil {
		return
	}

	return len(result.Rules), nil
}

func (l Listener) GetHighestPriority(elb *elbv2.Client) (highestPri int, err error) {
	result, err := elb.DescribeRules(context.TODO(), &elbv2.DescribeRulesInput{
		ListenerArn: aws.String(string(l)),
	})
	if err != nil {
		return
	}

	for _, rule := range result.Rules {
		priString := aws.ToString(rule.Priority)
		if priString == "default" {
			continue
		}
		pri, err := strconv.Atoi(priString)
		if err != nil {
			log.WithError(err).Notice("While paring priority as int")
			continue
		}
		if pri > highestPri {
			highestPri = pri
		}
	}

	return
}
