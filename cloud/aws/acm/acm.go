package acm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmTypes "github.com/aws/aws-sdk-go-v2/service/acm/types"

	log "github.com/cantara/bragi/sbragi"
)

type Cert struct {
	Id     string `json:"id"`
	Domain string `json:"name"`
	Status string
}

func GetCert(domain string, cm *acm.Client) (cert Cert, err error) {
	log.Trace("getting cert", "domain", domain)
	result, err := cm.ListCertificates(context.Background(), &acm.ListCertificatesInput{})
	if err != nil {
		err = fmt.Errorf("Unable to describe certificate, err: %v error:%v", err, ErrNoCertFound)
		return
	}

	for _, c := range result.CertificateSummaryList {
		log.Trace("testign cert", "name", domain, "found", *c.DomainName)
		if *c.DomainName != domain {
			continue
		}
		log.Trace("found cert", "cert", c)
		return Cert{
			Id:     *c.CertificateArn,
			Domain: domain,
			Status: string(c.Status),
		}, nil
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

func GetDomainValidation(arn string, cm *acm.Client) (key, val string, err error) {
	result, err := cm.DescribeCertificate(context.TODO(), &acm.DescribeCertificateInput{
		CertificateArn: &arn,
	})
	if err != nil {
		return
	}
	if result.Certificate.Status == acmTypes.CertificateStatusIssued {
		return
	}
	if len(result.Certificate.DomainValidationOptions) == 0 {
		err = errors.New("no domain validation options found")
		return
	}
	if result.Certificate.DomainValidationOptions[0].ResourceRecord == nil {
		err = errors.New("domain validation resource record not present yet")
		return
	}
	log.Debug("reading domain validation", "options", result.Certificate.DomainValidationOptions[0])
	record := *result.Certificate.DomainValidationOptions[0].ResourceRecord
	key, val = *record.Name, *record.Value
	log.Trace("dns validation found", "name", key, "val", val)
	return
}

func NewCert(name string, cm *acm.Client) (cert Cert, err error) {
	cert, err = GetCert(name, cm)
	if err != nil {
		if !errors.Is(err, ErrNoCertFound) {
			return Cert{}, err
		}
	} else {
		log.Trace("cert exists", "name", name)
		return cert, nil
	}
	log.Trace("creating new cert", "name", name)

	result, err := cm.RequestCertificate(context.Background(), &acm.RequestCertificateInput{
		DomainName:       &name,
		KeyAlgorithm:     acmTypes.KeyAlgorithmRsa2048, //No other type is listable???
		ValidationMethod: acmTypes.ValidationMethodDns,
	})

	if err != nil {
		return Cert{}, err
	}

	if result == nil {
		return Cert{}, fmt.Errorf("result was nil when creating cert %s", name)
	}

	id := *result.CertificateArn
	log.Trace("created cert", "id", id, "name", name, "result", result)

	return Cert{Id: id, Domain: name}, nil
}

func WaitUntilIssues(arn string, cm *acm.Client) error {
	return acm.NewCertificateValidatedWaiter(cm).Wait(context.TODO(), &acm.DescribeCertificateInput{
		CertificateArn: &arn,
	}, time.Minute*5)
}

/*
		_, err = e2.ModifyVpcAttribute(context.Background(), &ec2.ModifyVpcAttributeInput{
			VpcId:            &id,
			EnableDnsSupport: &ec2types.AttributeBooleanValue{Value: aws.Bool(true)},
		})
		_, err = e2.ModifyVpcAttribute(context.Background(), &ec2.ModifyVpcAttributeInput{
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
