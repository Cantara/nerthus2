package loadbalancer

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"

	//"github.com/aws/aws-sdk-go-v2/aws/awserr"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type TargetGroup struct {
	Name    string `json:"name"`
	UriPath string `json:"path"`
	Port    int    `json:"port"`
	ARN     string `json:"arn"`
}

func CreateTargetGroupName(scope, name string) (string, error) {
	tgName := strings.Split(scope, "-")[0] + "-" + strings.ToLower(name)
	tgName = strings.TrimSuffix(tgName, "api")
	tgName = strings.ReplaceAll(tgName, "-", " ")
	tgName = strings.TrimSpace(tgName)
	tgName = strings.ReplaceAll(tgName, " ", "-")
	tgName = tgName + "-tg"
	if len(tgName) > 32 {
		return "", fmt.Errorf("Calculated targetgroup name (%s) is to long based on input scope (%s) and name (%s). Max len 32.",
			tgName, scope, name)
	}
	return tgName, nil
}

func GetTargetGroup(name, uriPath string, port int, elb *elbv2.Client) (tg TargetGroup, err error) {
	result, err := elb.DescribeTargetGroups(context.TODO(), &elbv2.DescribeTargetGroupsInput{
		Names: []string{
			name,
		},
	})
	if err != nil {
		return
	}

	tg = TargetGroup{
		Name:    name,
		UriPath: uriPath,
		Port:    port,
	}
	tg.ARN = aws.ToString(result.TargetGroups[0].TargetGroupArn)
	return
}

func CreateTargetGroup(vpc, name, uriPath string, port int, elb *elbv2.Client) (TargetGroup, error) {
	result, err := elb.CreateTargetGroup(context.TODO(), &elbv2.CreateTargetGroupInput{
		Name:                       aws.String(name),
		Port:                       aws.Int32(int32(port)),
		Protocol:                   "HTTP",
		VpcId:                      aws.String(vpc),
		TargetType:                 "instance",
		ProtocolVersion:            aws.String("HTTP1"),
		HealthCheckIntervalSeconds: aws.Int32(5),
		HealthCheckPath:            aws.String(fmt.Sprintf("/%s/health", uriPath)), //FIXME: This is shady
		HealthCheckPort:            aws.String("traffic-port"),
		HealthCheckProtocol:        "HTTP",
		HealthCheckTimeoutSeconds:  aws.Int32(2),
		HealthyThresholdCount:      aws.Int32(2),
	})
	if err != nil {
		return TargetGroup{}, err
	}
	return TargetGroup{
		Name:    name,
		UriPath: uriPath,
		Port:    port,
		ARN:     aws.ToString(result.TargetGroups[0].TargetGroupArn),
	}, nil
}

func (tg *TargetGroup) Delete(elb *elbv2.Client) (err error) {
	_, err = elb.DeleteTargetGroup(context.TODO(), &elbv2.DeleteTargetGroupInput{
		TargetGroupArn: aws.String(tg.ARN),
	})
	return
}
