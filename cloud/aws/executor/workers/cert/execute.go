package cert

import (
	"errors"
	"fmt"

	log "github.com/cantara/bragi/sbragi"

	awsacm "github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/cantara/nerthus2/cloud/aws/acm"
	"github.com/cantara/nerthus2/cloud/aws/dns"
	"github.com/cantara/nerthus2/cloud/aws/executor"
)

type Requireing interface {
	Cert(acm.Cert) executor.Func
}

type data struct {
	ac   *awsacm.Client
	rc   *route53.Client
	base string
	name string
	rs   []Requireing
}

func Executor(base, env string, rs []Requireing, ac *awsacm.Client, rc *route53.Client) data {
	/*
		domainBaseName := "quadim.dev"
		domainName := fmt.Sprintf("dev.%s", domainBaseName)
		certName := fmt.Sprintf("*.%s", domainName)
	*/
	return data{
		ac:   ac,
		rc:   rc,
		base: base,
		name: fmt.Sprintf("*.%s.%s", env, base),
		rs:   rs,
	}
}

func (d data) Execute(c chan<- executor.Func) { //This almost needs it's own state machine based on all the possible errors here
	log.Trace("executing cert")

	cert, err := acm.GetCert(d.name, d.ac)
	if err == nil && cert.Status == string(types.CertificateStatusIssued) {
		for _, r := range d.rs {
			f := r.Cert(cert)
			if f == nil {
				continue
			}
			c <- f
		}
		return
	}
	if err != nil && !errors.Is(err, acm.ErrNoCertFound) {
		log.WithError(err).Error("while getting cert")
		c <- d.Execute
		return
	}
	cert, err = acm.NewCert(d.name, d.ac)
	if err != nil {
		log.WithError(err).Error("while creating cert")
		c <- d.Execute
		return
	}
	key, val, err := acm.GetDomainValidation(cert.Id, d.ac)
	if err != nil {
		log.WithError(err).Error("while getting cert domain validation")
		c <- d.Execute
		return
	}
	zone, err := dns.GetHostedZoneId(d.base, d.rc)
	if err != nil {
		log.WithError(err).Error("while getting cert domain hosted zone")
		c <- d.Execute
		return
	}
	err = dns.NewRecord(zone, key, val, "cname", d.rc)
	if err != nil {
		log.WithError(err).Error("while creating domain validation record")
		c <- d.Execute
		return
	}
	err = acm.WaitUntilIssues(cert.Id, d.ac)
	if err != nil {
		log.WithError(err).Error("while waiting for cert to be valid")
		c <- d.Execute
		return
	}

	cert, err = acm.GetCert(d.name, d.ac)
	if err != nil {
		log.WithError(err).Error("while getting cert")
		c <- d.Execute
		return
	}
	for _, r := range d.rs {
		f := r.Cert(cert)
		if f == nil {
			continue
		}
		c <- f
	}
}
