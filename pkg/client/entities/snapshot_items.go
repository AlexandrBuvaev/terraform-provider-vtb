package entities

func (c SnapshotItemConfig) GetProviderType() (string, string) {
	return "openstack", "snapshot"
}

type SnapshotItemConfig struct {
	Name       string   `json:"name"`
	Volumes    []string `json:"volumes"`
	DeleteDate string   `json:"delete_date"`
}
