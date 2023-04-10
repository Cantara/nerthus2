package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"os"
	"strings"
)

type Server struct {
	Host  string   `json:"host"`
	Name  string   `json:"name"`
	Users []string `json:"users"`
}

func GetServers() (servers []Server, err error) {
	var opts []func(*config.LoadOptions) error
	if os.Getenv("aws.profile") != "" {
		opts = append(opts, config.WithSharedConfigProfile(os.Getenv("aws.profile")))
	} else {
		opts = append(opts, config.WithDefaultRegion(os.Getenv("aws.region")))
	}
	sess, err := config.LoadDefaultConfig(context.TODO(), opts...)
	e2 := ec2.NewFromConfig(sess)
	result, err := e2.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name: aws.String("tag:Manager"),
				Values: []string{
					"nerthus",
				},
			},
		},
	})
	if err != nil {
		return
	}
	if len(result.Reservations) < 1 {
		err = fmt.Errorf("No servers managed by nerthus")
		return
	}
	/* if len(result.Reservations) > 1 {
		err = fmt.Errorf("Too many servers with name %s", name)
	} */
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.State.Name != ec2types.InstanceStateNameRunning {
				continue
			}
			var serverName string
			var usernames []string
			for _, tag := range instance.Tags {
				if aws.ToString(tag.Key) == "Name" {
					serverName = *tag.Value
				} else if aws.ToString(tag.Key) == "OS" {
					if strings.HasPrefix(strings.ToLower(*tag.Value), "amazon") {
						usernames = []string{"ec2-user"}
					} else if strings.HasPrefix(strings.ToLower(*tag.Value), "ubuntu") {
						usernames = []string{"ubuntu"}
					} else if strings.HasPrefix(strings.ToLower(*tag.Value), "debian") {
						usernames = []string{"admin"}
					} else if strings.HasPrefix(strings.ToLower(*tag.Value), "centos") {
						usernames = []string{"centos", "ec2-user"}
					} else if strings.HasPrefix(strings.ToLower(*tag.Value), "fedora") {
						usernames = []string{"fedora", "ec2-user"}
					}
				} else if aws.ToString(tag.Key) == "Services" {
					usernames = append(usernames, strings.Split(*tag.Value, ",")...)
				}
			}
			servers = append(servers, Server{
				Host:  aws.ToString(instance.PublicIpAddress),
				Name:  serverName,
				Users: usernames,
			})
		}
	}
	return
}
