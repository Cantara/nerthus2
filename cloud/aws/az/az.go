package az

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	log "github.com/cantara/bragi/sbragi"
)

var azCache []string

func GetAZs(e2 *ec2.Client) (azs []string, err error) {
	if len(azCache) > 0 {
		return azCache, nil
	}
	log.Trace("getting availability zones")
	result, err := e2.DescribeAvailabilityZones(context.Background(), &ec2.DescribeAvailabilityZonesInput{
		/* TODO: explore why this does not work
		Filters: []ec2types.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []string{
					name,
				},
			},
		},
		*/
	})
	if err != nil {
		err = fmt.Errorf("Unable to describe VPCs, err: %v", err)
		return
	}
	if len(result.AvailabilityZones) == 0 {
		log.Trace("no Availability zones found")
		err = fmt.Errorf("no availability zones found")
		return
	}

	for _, az := range result.AvailabilityZones {
		if *az.ZoneType != "availability-zone" {
			continue
		}
		if az.State != "available" {
			continue
		}
		//azs = append(azs, *az.ZoneName)
		azs = append(azs, *az.ZoneId)
	}

	sort.Strings(azs)
	azCache = azs
	return
}
