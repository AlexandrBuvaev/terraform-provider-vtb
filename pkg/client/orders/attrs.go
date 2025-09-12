package orders

import (
	"terraform-provider-vtb/pkg/client/entities"
)

// BasicAttrs mostly common fields almost for every product in order-service
type BasicAttrs struct {
	ADIntegration       bool                     `json:"ad_integration"`
	ADLogonGrants       []entities.ADLogonGrants `json:"ad_logon_grants"`
	Domain              string                   `json:"domain"`
	DefaultNic          entities.DefaultNic      `json:"default_nic"`
	Flavor              entities.Flavor          `json:"flavor"`
	ExtraMounts         []entities.ExtraMount    `json:"extra_mounts"`
	Platform            string                   `json:"platform"`
	OnSupport           bool                     `json:"on_support"`
	OsVersion           string                   `json:"os_version,omitempty"`
	ImageID             string                   `json:"image_id,omitempty"` // Для продукта Linux preprom(УСИ)
	AvailabilityZone    string                   `json:"availability_zone"`
	DataCenter          string                   `json:"data_center,omitempty"`
	CreatedWithOpenTofu bool                     `json:"created_with_opentofu"`
}
