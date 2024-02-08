package loadbalancer

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	log "github.com/cantara/bragi/sbragi"

	//"github.com/aws/aws-sdk-go-v2/aws/awserr"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type Target struct {
	targetGroup TargetGroup
	server      string
}

func CreateTarget(tg TargetGroup, s string, elb *elbv2.Client) (t Target, err error) {
	log.Trace("creating target", "tg", tg.ARN, "node", s)
	_, err = elb.RegisterTargets(context.Background(), &elbv2.RegisterTargetsInput{
		TargetGroupArn: aws.String(tg.ARN),
		Targets: []elbv2types.TargetDescription{
			{
				Id: aws.String(s),
			},
		},
	})
	if err != nil {
		return
	}
	t = Target{
		targetGroup: tg,
		server:      s,
	}
	return
}

func (t *Target) Delete(elb *elbv2.Client) (err error) {
	_, err = elb.DeregisterTargets(context.Background(), &elbv2.DeregisterTargetsInput{
		TargetGroupArn: aws.String(t.targetGroup.ARN),
		Targets: []elbv2types.TargetDescription{
			{
				Id: aws.String(t.server),
			},
		},
	})
	return
}
