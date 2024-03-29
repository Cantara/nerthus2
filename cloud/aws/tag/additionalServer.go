package tag

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cantara/nerthus2/cloud/aws/util"
)

type additional struct {
	Scope              string `json:"-"`
	Name               string `json:"name"`
	ServerId           string `json:"server_id"`
	VolumeId           string `json:"volume_id"`
	NetworkInterfaceId string `json:"network_interface_id"`
	ImageId            string `json:"image_id"`
	tag                *tag
	ec2                *ec2.Client
	created            bool
}

func NewAddTag(serviceName, scope, serverId, networkInterfaceId, volumeId, imageId string, e2 *ec2.Client) (t additional, err error) {
	err = util.CheckEC2Session(e2)
	if err != nil {
		return
	}
	t = additional{
		Scope:              scope,
		Name:               serviceName,
		ServerId:           serverId,
		VolumeId:           volumeId,
		NetworkInterfaceId: networkInterfaceId,
		ImageId:            imageId,
		tag: &tag{
			ec2Resources: []string{
				serverId,
				networkInterfaceId,
				volumeId,
				//aws.String(t.ImageId),
			},
			Key:   serviceName,
			Value: scope,
			ec2:   e2,
		},
	}
	return
}

func (t *additional) Create() (id string, err error) {
	id, err = t.tag.Create()
	return
}

func (t *additional) Delete() (err error) {
	err = t.tag.Delete()
	return
}
