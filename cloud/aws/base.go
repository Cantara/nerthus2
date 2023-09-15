package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/cloud/aws/key"
	"github.com/cantara/nerthus2/cloud/aws/security"
	serverlib "github.com/cantara/nerthus2/cloud/aws/server"
	"github.com/cantara/nerthus2/cloud/aws/util"
	"github.com/cantara/nerthus2/slack"
)

func CheckNameLen(name string) error {
	const maxNameLen = 29
	const minNameLen = 3
	if len(name) < minNameLen {
		return fmt.Errorf("Min name len is: %d provided name len is %d.", minNameLen, len(name))
	}
	if len(name) > maxNameLen {
		return fmt.Errorf("Max name len in aws is: %d provided name len is %d.", maxNameLen, len(name))
	}
	return nil
}

type AWS struct {
	ec2 *ec2.Client
	elb *elbv2.Client
	rds *rds.Client
}

func (a AWS) GetEC2() *ec2.Client {
	return a.ec2
}

func (a AWS) GetELB() *elbv2.Client {
	return a.elb
}

func (a AWS) GetRDS() *rds.Client {
	return a.rds
}

func (a *AWS) NewEC2(c aws.Config) {
	if a.ec2 != nil {
		return
	}
	a.ec2 = ec2.NewFromConfig(c)
}

func (a *AWS) NewELB(c aws.Config) {
	if a.elb != nil {
		return
	}
	a.elb = elbv2.NewFromConfig(c)
}

func (a *AWS) NewRDS(c aws.Config) {
	if a.rds != nil {
		return
	}
	a.rds = rds.NewFromConfig(c)
}

func cleanup(object, logMessage string, obj util.AWSObject) func() {
	return func() {
		s := fmt.Sprintf(" Cleaning up: %s", object)
		log.Info(s)
		slack.SendStatus(s)
		err := obj.Delete()
		if err != nil {
			log.AddError(err).Crit(logMessage)
		}
	}
}

func NewStack() Stack {
	return Stack{}
}

type Stack struct {
	funcs []func()
}

func (s *Stack) Push(fun func()) {
	s.funcs = append(s.funcs, fun)
}

func (s *Stack) Pop() func() {
	if s.Empty() {
		return nil
	}
	fun := s.Last()
	s.funcs = s.funcs[:s.Len()-1]
	return fun
}

func (s Stack) Len() int {
	return len(s.funcs)
}

func (s Stack) Last() func() {
	if s.Empty() {
		return nil
	}
	return s.funcs[s.Len()-1]
}

func (s Stack) First() func() {
	if s.Empty() {
		return nil
	}
	return s.funcs[0]
}

func (s Stack) Empty() bool {
	return s.Len() == 0
}

func GetPublicDNS(server, scope string, a *AWS) (publicDNS string, err error) {
	serv, err := serverlib.GetServer(server, scope, key.Key{}, security.Group{}, a.ec2)
	if err != nil {
		return
	}
	publicDNS = serv.PublicDNS
	return
}
