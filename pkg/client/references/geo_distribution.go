package references

import (
	"encoding/json"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
)

type GeoDistributor interface {
	GetGeoResponseType() string
}

type DataCenter struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type GeoStructure struct {
	Dc         string     `json:"dc"`
	Category   string     `json:"category"`
	Quantity   int        `json:"quantity"`
	DataCenter DataCenter `json:"data_center"`
}

type KafkaGeoPage struct {
	ReferenceResponse
	Data struct {
		Kafka     []GeoStructure `json:"kafka"`
		Zookeeper []GeoStructure `json:"zookeeper"`
	} `json:"data"`
}

func (g KafkaGeoPage) GetGeoResponseType() string {
	return "kafka"
}

func GetKafkaGeoPageByID(
	creds *auth.Credentials,
	label,
	id,
	organization,
	environment string,
) (*KafkaGeoPage, error) {

	tags := fmt.Sprintf(
		"%s,%s,%s",
		strings.ToUpper(environment),
		label,
		strings.ToLower(organization),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "geo_distribution", parameters)
	if err != nil {
		return nil, err
	}

	if label == "kafka" {
		var refPages []KafkaGeoPage
		err = json.Unmarshal(body, &refPages)
		if err != nil {
			return nil, err
		}
		for _, page := range refPages {
			if page.ID == id {
				return &page, nil
			}
		}
	} else {
		return nil, fmt.Errorf("can't find type of `geo_distribution` for type='%s'", label)
	}

	return nil, fmt.Errorf("can't find page in `geo_distribution` reference with tags=%s", tags)
}

func GetKafkaGeoPage(
	creds *auth.Credentials,
	label,
	referenceName,
	organization,
	environment string,
) (*KafkaGeoPage, error) {

	parameters := map[string]string{
		"tags__contains": fmt.Sprintf(
			"%s,%s,%s",
			strings.ToUpper(environment),
			label,
			strings.ToLower(organization),
		),
	}

	body, err := getReferenceData(creds.AccessToken, "geo_distribution", parameters)
	if err != nil {
		return nil, err
	}

	if label == "kafka" {
		var objects []KafkaGeoPage
		err = json.Unmarshal(body, &objects)
		if err != nil {
			return nil, err
		}
		for _, record := range objects {
			if record.Name == referenceName {
				return &record, nil
			}
		}
	} else {
		return nil, fmt.Errorf("find find type of geo distribution for type = '%s'", label)
	}

	return nil, fmt.Errorf(
		"can't find record in geo distribution with parameters: (type:'%s', reference_name: '%s', organiztion:'%s', environment:'%s')", label, referenceName, organization, environment,
	)
}

func GetGeoPage(creds *auth.Credentials, tags ...string) (*ReferenceResponse, error) {
	parameters := map[string]string{"tags__contains": strings.Join(tags, ",")}
	body, err := getReferenceData(creds.AccessToken, "geo_distribution", parameters)
	if err != nil {
		return nil, err
	}

	var objects []ReferenceResponse
	err = json.Unmarshal(body, &objects)
	if err != nil {
		return nil, err
	}

	if len(objects) > 1 {
		return nil, fmt.Errorf("can't get page. Too many records in geo discribution response")
	}
	return &objects[0], nil
}

func GetGeoPageByLayout(creds *auth.Credentials, layoutName string) (*ReferenceResponse, error) {
	parameters := map[string]string{"name": layoutName}
	body, err := getReferenceData(creds.AccessToken, "geo_distribution", parameters)
	if err != nil {
		return nil, err
	}

	var objects []ReferenceResponse
	err = json.Unmarshal(body, &objects)
	if err != nil {
		return nil, err
	}

	if len(objects) > 1 {
		return nil, fmt.Errorf("can't get page. Too many records in geo discribution response")
	}
	return &objects[0], nil
}

func GetGeoDistributionLayoutID(creds *auth.Credentials, layout string, tags ...string) (string, error) {

	parameters := map[string]string{"tags__contains": strings.Join(tags, ",")}
	body, err := getReferenceData(creds.AccessToken, "geo_distribution", parameters)
	if err != nil {
		return "", err
	}

	var pages []ReferenceResponse
	err = json.Unmarshal(body, &pages)
	if err != nil {
		return "", err
	}

	var availableLayouts []string
	for _, page := range pages {
		if page.Name == layout {
			return page.ID, nil
		}
		availableLayouts = append(availableLayouts, fmt.Sprint(page.Name))
	}

	return "", fmt.Errorf(
		"geo_distribution error: tags=%s, layout=%s not found\nAvailable: %s",
		tags, layout, availableLayouts,
	)
}

func GetGeoDistributionLayoutNameByID(creds *auth.Credentials, layoutID string, tags ...string) (string, error) {

	parameters := map[string]string{"tags__contains": strings.Join(tags, ",")}
	body, err := getReferenceData(creds.AccessToken, "geo_distribution", parameters)
	if err != nil {
		return "", err
	}

	var pages []ReferenceResponse
	err = json.Unmarshal(body, &pages)
	if err != nil {
		return "", err
	}

	var availableLayouts []string
	for _, page := range pages {
		if page.ID == layoutID {
			return page.Name, nil
		}
		availableLayouts = append(availableLayouts, fmt.Sprint(page.Name))
	}

	return "", fmt.Errorf(
		"geo_distribution error: tags=%s, layout_id=%s not found\nAvailable: %s",
		tags, layoutID, availableLayouts,
	)
}
