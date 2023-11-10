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
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/fairytale/adapter"
	"github.com/cantara/nerthus2/cloud/aws/executor/workers/start"
)

type inn struct {
	Base string `json:"base"`
}

var Fingerprint = adapter.New[acm.Cert]("CreateOrGetCert")

func Adapter(ac *awsacm.Client, rc *route53.Client) adapter.Adapter {
	return Fingerprint.Adapter(func(a []adapter.Value) (cert acm.Cert, err error) {
		//base, _ := adapter.Value[string](a[0])
		i := start.Fingerprint.Value(a[0])
		name := fmt.Sprintf("*.%s", i.Base)
		log.Trace("executing cert")

		cert, err = acm.GetCert(name, ac)
		if err == nil && cert.Status == string(types.CertificateStatusIssued) {
			return cert, nil
		}
		if err != nil && !errors.Is(err, acm.ErrNoCertFound) {
			log.WithError(err).Error("while getting cert")
			return
		}
		cert, err = acm.NewCert(name, ac)
		if err != nil {
			log.WithError(err).Error("while creating cert")
			return
		}
		key, val, err := acm.GetDomainValidation(cert.Id, ac)
		if err != nil {
			log.WithError(err).Error("while getting cert domain validation")
			return
		}
		zone, err := dns.GetHostedZoneId(i.Base, rc)
		if err != nil {
			log.WithError(err).Error("while getting cert domain hosted zone")
			return
		}
		err = dns.NewRecord(zone, key, val, "cname", rc)
		if err != nil {
			log.WithError(err).Error("while creating domain validation record")
			return
		}
		err = acm.WaitUntilIssues(cert.Id, ac)
		if err != nil {
			log.WithError(err).Error("while waiting for cert to be valid")
			return
		}

		cert, err = acm.GetCert(name, ac)
		if err != nil {
			log.WithError(err).Error("while getting cert")
			return
		}
		return cert, nil
	}, start.Fingerprint)
}

type data struct {
	ac   *awsacm.Client
	rc   *route53.Client
	base string
	name string
}

func Executor(base string, ac *awsacm.Client, rc *route53.Client) data {
	/*
		domainBaseName := "quadim.dev"
		domainName := fmt.Sprintf("dev.%s", domainBaseName)
		certName := fmt.Sprintf("*.%s", domainName)
	*/
	return data{
		ac:   ac,
		rc:   rc,
		base: base,
		name: fmt.Sprintf("*.%s", base),
	}
}

func (d data) Execute() (cert acm.Cert, err error) { //This almost needs it's own state machine based on all the possible errors here
	log.Trace("executing cert")

	cert, err = acm.GetCert(d.name, d.ac)
	if err == nil && cert.Status == string(types.CertificateStatusIssued) {
		return
	}
	if err != nil && !errors.Is(err, acm.ErrNoCertFound) {
		log.WithError(err).Error("while getting cert")
		return
	}
	cert, err = acm.NewCert(d.name, d.ac)
	if err != nil {
		log.WithError(err).Error("while creating cert")
		return
	}
	key, val, err := acm.GetDomainValidation(cert.Id, d.ac)
	if err != nil {
		log.WithError(err).Error("while getting cert domain validation")
		return
	}
	zone, err := dns.GetHostedZoneId(d.base, d.rc)
	if err != nil {
		log.WithError(err).Error("while getting cert domain hosted zone")
		return
	}
	err = dns.NewRecord(zone, key, val, "cname", d.rc)
	if err != nil {
		log.WithError(err).Error("while creating domain validation record")
		return
	}
	err = acm.WaitUntilIssues(cert.Id, d.ac)
	if err != nil {
		log.WithError(err).Error("while waiting for cert to be valid")
		return
	}

	cert, err = acm.GetCert(d.name, d.ac)
	if err != nil {
		log.WithError(err).Error("while getting cert")
		return
	}
	return
}
