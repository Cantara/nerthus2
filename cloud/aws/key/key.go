package key

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	log "github.com/cantara/bragi/sbragi"
)

type Key struct {
	Cluster     string           `json:"-"`
	Id          string           `json:"id"`
	Name        string           `json:"name"`
	PemName     string           `json:"pem_name"`
	Fingerprint string           `json:"fingerprint"`
	Material    string           `json:"material"`
	Type        ec2types.KeyType `json:"type"`
	Created     time.Time
}

func New(cluster string, e2 *ec2.Client) (k Key, err error) {
	keyName := cluster + "-key"
	k, err = Get(keyName, e2)
	if err != nil && !errors.Is(err, ErrNoKeyFound) {
		log.WithoutEscalation().WithError(err).Trace("get error")
		return
	}
	if k.Id != "" {
		log.Trace("key exists")
		return
	}
	k = Key{
		Cluster: cluster,
		Name:    keyName,
		Type:    ec2types.KeyTypeEd25519,
	}
	keyResult, err := e2.CreateKeyPair(context.Background(), &ec2.CreateKeyPairInput{
		KeyName: aws.String(k.Name),
		KeyType: k.Type,
	})
	if err != nil {
		return
	}
	k.Id = aws.ToString(keyResult.KeyPairId)
	k.Fingerprint = aws.ToString(keyResult.KeyFingerprint)
	k.Material = aws.ToString(keyResult.KeyMaterial)
	k.PemName = k.Name + ".pem"
	k.Created = time.Now()
	return
}

func Get(name string, e2 *ec2.Client) (k Key, err error) {
	log.Trace("getting key", "name", name)

	result, err := e2.DescribeKeyPairs(context.Background(), &ec2.DescribeKeyPairsInput{
		KeyNames: []string{
			name,
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "InvalidKeyPair.NotFound") {
			err = errors.Join(ErrNoKeyFound, err)
		}
		return
	}
	if len(result.KeyPairs) < 1 {
		err = fmt.Errorf("No key with name %s", name)
		return
	}
	kp := result.KeyPairs[0]
	k = Key{
		Id:          *kp.KeyPairId,
		Fingerprint: *kp.KeyFingerprint,
		PemName:     *kp.KeyName + ".pem",
		Name:        *kp.KeyName,
		Created:     *kp.CreateTime,
	}
	return
}

func Wait(k Key, e2 *ec2.Client) (err error) {
	err = ec2.NewKeyPairExistsWaiter(e2).Wait(context.Background(), &ec2.DescribeKeyPairsInput{
		KeyPairIds: []string{
			k.Id,
		},
	}, 5*time.Minute)
	return
}

var ErrNoKeyFound = fmt.Errorf("no key found")

/*
func (k *Key) Delete() (err error) {
	if !k.created {
		return
	}
	err = util.CheckEC2Session(k.ec2)
	if err != nil {
		return
	}
	input := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(k.Name),
	}

	_, err = k.ec2.DeleteKeyPair(context.Background(), input)
	return
}

func (k Key) WithEC2(e *ec2.Client) Key {
	k.ec2 = e
	return k
}
*/
