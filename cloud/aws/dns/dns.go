package dns

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"

	log "github.com/cantara/bragi/sbragi"
)

type Cert struct {
	Id     string `json:"id"`
	Domain string `json:"name"`
}

func GetHostedZoneId(domain string, r53 *route53.Client) (id string, err error) {
	log.Trace("getting hosted zone", "domain", domain)
	domain = domain + "."
	result, err := r53.ListHostedZones(context.TODO(), &route53.ListHostedZonesInput{
		//result, err := r53.ListHostedZonesByName(context.TODO(), &route53.ListHostedZonesByNameInput{
		//DNSName: &domain,
	})
	if err != nil {
		err = fmt.Errorf("Unable to list hosted zones, err: %v", err)
		return
	}

	for _, zone := range result.HostedZones {
		log.Trace("testing hosted zone", "name", *zone.Name)
		if *zone.Name != domain {
			continue //Should probably just return error here as it is already ordered
		}
		id := *zone.Id
		idParts := strings.Split(id, "/")
		id = idParts[len(idParts)-1]
		log.Trace("found zone", "zone", zone, "id", id)
		return id, nil
	}
	/*
		arn := aws.ToString(result.Certificate.CertificateArn)
		log.Trace("checking cert", "domain", domain, "found", arn)
		dn := aws.ToString(result.Certificate.DomainName)
		if dn != domain {
			err = fmt.Errorf("certificate domain naim does not match %s != %s, error:%v", dn, domain, ErrNoCertFound)
			return
		}
		log.Trace("found cert", "domain", domain, "id", arn)
		cert = Cert{
			Id:     arn,
			Domain: dn,
		}
	*/
	err = ErrNoCertFound
	return
}

func ToRecordType(t string) (route53types.RRType, error) {
	t = strings.ToLower(t)
	for _, rt := range route53types.RRTypeA.Values() {
		if strings.ToLower(string(rt)) == t {
			return rt, nil
		}
	}
	return route53types.RRType(""), fmt.Errorf("invalid record type, %s", t)
}

func NewRecord(zone, key, val, t string, r53 *route53.Client) (err error) {
	if key == "" {
		return
	}
	rt, err := ToRecordType(t)
	if err != nil {
		return
	}
	/*
		record, err := GetRecord(zone, key, r53)
		if err != nil {
			if !errors.Is(err, ErrNoCertFound) {
				return err
			}
		} else {
			log.Trace("record exists", "zone", zone, "key", key)
			return nil
		}
	*/
	log.Trace("creating new record", "zone", zone, "key", key)

	_, err = r53.ChangeResourceRecordSets(context.TODO(), &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53types.ChangeBatch{
			Changes: []route53types.Change{
				{
					Action: route53types.ChangeActionCreate,
					ResourceRecordSet: &route53types.ResourceRecordSet{
						Name: aws.String(key),
						ResourceRecords: []route53types.ResourceRecord{
							{
								Value: aws.String(val),
							},
						},
						TTL:  aws.Int64(60),
						Type: rt,
					},
				},
			},
		},
		HostedZoneId: aws.String(zone),
	})

	if err != nil {
		return err
	}

	return
}

/*
		_, err = e2.ModifyVpcAttribute(context.TODO(), &ec2.ModifyVpcAttributeInput{
			VpcId:            &id,
			EnableDnsSupport: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		_, err = e2.ModifyVpcAttribute(context.TODO(), &ec2.ModifyVpcAttributeInput{
			VpcId:              &id,
			EnableDnsHostnames: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
		})

		vpc = VPC{
			Id:     id,
			Name:   name,
			CIDR:   cidr,
			CIDRv6: ipv6Cidr,
		}

		return
	}

	func Tag(tags []ec2types.Tag, key string) string {
		for _, tag := range tags {
			if strings.ToLower(aws.ToString(tag.Key)) != key {
				continue
			}
			return aws.ToString(tag.Value)
		}
		return ""
*/

var ErrNoCertFound = fmt.Errorf("no cert found")
