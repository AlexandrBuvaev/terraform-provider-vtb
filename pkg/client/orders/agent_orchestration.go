package orders

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type AgentOrchestration struct {
	Order
	Attrs AgentOrchestrationAttrs `json:"attrs"`
}

type SferaAgentAttrs struct {
	JenkinsAgentInstance  string                         `json:"jenkins_agent_instance"`
	JenkinsAgentExecutors int64                          `json:"jenkins_agent_executors"`
	JenkinsAgentSubsystem entities.JenkinsAgentSubsystem `json:"jenkins_agent_subsystem"`
}

type AgentOrchestrationAttrs struct {
	ADIntegration       bool                  `json:"ad_integration"`
	Domain              string                `json:"domain"`
	DefaultNic          entities.DefaultNic   `json:"default_nic"`
	Flavor              entities.Flavor       `json:"flavor"`
	ExtraMounts         []entities.ExtraMount `json:"extra_mounts"`
	Platform            string                `json:"platform"`
	OnSupport           bool                  `json:"on_support"`
	OsVersion           string                `json:"os_version,omitempty"`
	ImageID             string                `json:"image_id,omitempty"` // Для продукта Linux preprom(УСИ)
	AvailabilityZone    string                `json:"availability_zone"`
	DataCenter          string                `json:"data_center,omitempty"`
	SferaAgent          SferaAgentAttrs       `json:"sfera_agent"`
	CreatedWithOpenTofu bool                  `json:"created_with_opentofu"`
}

func NewAgentOrchestration(creds *auth.Credentials, projectName, productID string, attrs AgentOrchestrationAttrs) *AgentOrchestration {
	return &AgentOrchestration{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *AgentOrchestration) Create(p CreateOrderPayload) error {

	payload, err := o.createPayload(p, o)
	if err != nil {
		return err
	}

	resp, err := requests.SendRequest(o.Creds.AccessToken, generateCreateOrderUri(o.ProjectName), "POST", payload, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	trimmedBody := strings.Trim(string(body), "[]")
	err = json.Unmarshal([]byte(trimmedBody), o)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}
	return nil
}

// Расширение точек монтирования
func (o *AgentOrchestration) ExpandMountPoint(mountPoint entities.ExtraMount) (err error) {

	if err = o.requiredState("on"); err != nil {
		return
	}

	vmItems, err := o.GetVMItems()
	if err != nil {
		return
	}

	stateOff := "off"
	stateOn := "on"

	if vmItems[0].Data.State != stateOn {
		err = fmt.Errorf(
			"can't do action, because vm if vm.state is not equal %s, current state is %s",
			stateOn, stateOff,
		)
		return
	}

	extraMount, err := o.GetExtraMount(mountPoint.Path)
	if err != nil {
		return err
	}

	if extraMount == nil {
		err = fmt.Errorf("can't find mount point with path = %s", mountPoint.Path)
		return
	}

	if mountPoint == *extraMount {
		return
	}

	increaseSize := mountPoint.Size - extraMount.Size
	if increaseSize < 10 {
		err = fmt.Errorf(
			"can't change mount point less then 10 GB. Current is %d, new is %d",
			extraMount.Size, mountPoint.Size,
		)
		return
	}

	attrs := map[string]interface{}{
		"mount": mountPoint.Path,
		"size":  int(increaseSize),
	}

	data := map[string]interface{}{
		"item_id": vmItems[0].ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("expand_mount_point_new")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *AgentOrchestration) GetAttrs() interface{} {
	return o.Attrs
}

// Получение точек монтирования
func (o *AgentOrchestration) GetExtraMount(path string) (*entities.ExtraMount, error) {

	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	if len(items) != 1 {
		return nil, errors.New("VMs len count must be equal 1")
	}

	extraMounts := items[0].Data.Config.(entities.VMItemConfig).ExtraMounts
	if len(extraMounts) == 0 {
		return nil, errors.New("length should be more then 0")
	}

	for _, em := range extraMounts {
		if em.Mount == path {
			return &entities.ExtraMount{
				Path:       em.Mount,
				Size:       int64(em.Size),
				FileSystem: em.Fstype,
			}, nil
		}
	}
	return nil, nil
}

// Изменение конфигурации
func (o *AgentOrchestration) ChangeFlavor(flavor entities.Flavor) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"flavor":      flavor,
		"check_agree": true,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("two_layer_resize_vm_agent_orchestration")

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *AgentOrchestration) AgentDeleteTwoLayer() error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("delete_two_layer_agent_orchestration")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}

	return err
}
