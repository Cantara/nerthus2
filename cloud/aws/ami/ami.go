package ami

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	log "github.com/cantara/bragi/sbragi"
	"github.com/cantara/nerthus2/config/schema"
)

const DateFormat = "2006-01-02T15:04:05Z" //YYYY-MM-DDTHH:MM:SSZ

type Image struct {
	Id        string
	Name      string
	HName     string
	Arch      schema.Arch
	RootDev   string
	created   time.Time
	depreated time.Time
	got       time.Time
}

func nameToAmiName(s string) string {
	switch strings.ToLower(s) {
	case "amazon linux 2023 minimal":
		fallthrough
	case "amazon linux 2023 min":
		return "al2023-ami-minimal-2023.*"
	case "amazon linux 2023":
		return "al2023-ami-2023.*"
	case "amazon linux 2":
		return "amzn2-ami-kernel-*-hvm-2.*"
	}

	return fmt.Sprintf("%s-*", strings.ReplaceAll(strings.ToLower(s), " ", "-"))
}

func (img Image) Username() string {
	n := strings.ToLower(img.HName)
	if strings.HasPrefix(n, "amazon") {
		return "ec2-user"
	}
	if strings.HasPrefix(n, "ubuntu") {
		return "ubuntu"
	}
	return ""
}

var imageCache []Image //Needs syncronization

func GetImage(name string, arch schema.Arch, e2 *ec2.Client) (newest Image, err error) {
	log.Trace("getting image", "name", name, "arch", arch, "cache", imageCache)
	for i, img := range imageCache {
		if img.Arch != arch {
			continue
		}
		if img.HName != name {
			continue
		}
		if img.got.Add(time.Hour * 24).After(time.Now()) {
			return img, nil
		}
		imageCache = append(imageCache[:i], imageCache[i:]...)
	}

	result, err := e2.DescribeImages(context.Background(), &ec2.DescribeImagesInput{
		Owners: []string{
			"amazon",
		},
		Filters: []ec2types.Filter{
			{
				Name: aws.String("architecture"),
				Values: []string{
					arch.String(),
				},
			},
			{
				Name: aws.String("name"),
				Values: []string{
					nameToAmiName(name),
				},
			},
		},
	})
	if err != nil {
		return
	}
	if len(result.Images) < 1 {
		err = fmt.Errorf("No image with name %s", name)
		return
	}
	/* if len(result.Reservations) > 1 {
		err = fmt.Errorf("Too many servers with name %s", name)
	} */
	images := make([]Image, len(result.Images))
	for i, img := range result.Images {
		if !strings.HasPrefix(*img.Name, "a") {
			continue
		}
		log.Trace("Looking for image", "name", name, "found", *img.Name)
		arch, _ := schema.StringToArch(string(img.Architecture))
		created, _ := time.Parse(*img.CreationDate, DateFormat)
		depreated, _ := time.Parse(*img.DeprecationTime, DateFormat)
		images[i] = Image{
			Id:      *img.ImageId,
			Name:    *img.Name,
			Arch:    arch,
			RootDev: aws.ToString(img.RootDeviceName),
			//Username:
			created:   created,
			depreated: depreated,
		}
	}
	for _, img := range images {
		if img.Arch != arch {
			continue
		}
		if newest.created.After(img.created) {
			continue
		}
		newest = img
	}
	newest.got = time.Now()
	newest.HName = name
	imageCache = append(imageCache, newest)
	return
}
