package resource

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type HostedZoneLookup struct {
	*AwsResource
}

type HostedZoneLookupInputs struct {
	Domain string `json:"domain"`
}

type HostedZoneLookupOutputs struct {
	ZoneId string `json:"zoneId"`
}

func (r *HostedZoneLookup) Create(input *HostedZoneLookupInputs, output *CreateResult[HostedZoneLookupOutputs]) error {
	zoneId, err := r.lookup(input.Domain)
	if err != nil {
		return err
	}

	*output = CreateResult[HostedZoneLookupOutputs]{
		ID:   zoneId,
		Outs: HostedZoneLookupOutputs{ZoneId: zoneId},
	}
	return nil
}

func (r *HostedZoneLookup) Update(input *UpdateInput[HostedZoneLookupInputs, HostedZoneLookupOutputs], output *UpdateResult[HostedZoneLookupOutputs]) error {
	zoneId, err := r.lookup(input.News.Domain)
	if err != nil {
		return err
	}

	*output = UpdateResult[HostedZoneLookupOutputs]{
		Outs: HostedZoneLookupOutputs{ZoneId: zoneId},
	}
	return nil
}

func (r *HostedZoneLookup) lookup(domain string) (string, error) {
	cfg, err := r.config()
	if err != nil {
		return "", err
	}

	client := route53.NewFromConfig(cfg)

	var zones []types.HostedZone
	var nextMarker *string

	for {
		res, err := client.ListHostedZones(r.context, &route53.ListHostedZonesInput{
			Marker: nextMarker,
		})
		if err != nil {
			return "", err
		}

		zones = append(zones, res.HostedZones...)

		if res.NextMarker == nil {
			break
		}
		nextMarker = res.NextMarker
	}

	parts := strings.Split(domain, ".")
	for i := 0; i <= len(parts)-2; i++ {
		zoneName := strings.Join(parts[i:], ".") + "."
		for _, zone := range zones {
			if *zone.Name == zoneName {
				return strings.TrimPrefix(*zone.Id, "/hostedzone/"), nil
			}
		}
	}

	return "", fmt.Errorf("could not find hosted zone for domain %s", domain)
}
