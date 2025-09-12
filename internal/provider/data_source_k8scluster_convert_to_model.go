package provider

import (
	"context"
	"reflect"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/references"

	// "github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func ConvertRegionToModel(data entities.K8sClusterConfigRegions, isClusterRegion bool) K8sClusterRegionModel {
	var iscodes basetypes.ListValue
	if isClusterRegion {
		elements := make([]attr.Value, len(data.IsCodes))
		for i, code := range data.IsCodes {
			elements[i] = types.StringValue(code)
		}
		iscodes = basetypes.ListValue(types.SetValueMust(types.StringType, elements))
	} else {
		iscodes = types.ListNull(types.StringType)
	}
	// var regionsVMsList []types.String
	// for _, vm := range data.VMs {
	// 	regionsVMsList = append(regionsVMsList, types.StringValue(vm.Hostname))
	// }
	// var regionsVMsAttrValues []attr.Value
	// for _, vm := range regionsVMsList {
	// 	regionsVMsAttrValues = append(regionsVMsAttrValues, vm)
	// }
	// regionsVMs, _ := types.ListValue(types.StringType, regionsVMsAttrValues)
	return K8sClusterRegionModel{
		// VMs:  regionsVMs,
		Name: types.StringValue(data.Name),
		Size: types.Int64Value(data.Size),
		Flavor: FlavorModel{
			Cores:  types.Int64Value(data.Flavor.Cpus),
			Memory: types.Int64Value(data.Flavor.Memory),
			Name:   types.StringValue(data.Flavor.Name),
			UUID:   types.StringValue(data.Flavor.UUID),
		},
		IsCodes:              iscodes,
		PodCPUMax:            types.Int64Value(data.PodCPUMax),
		PodMemoryMax:         types.Int64Value(data.PodMemoryMax),
		ContainerCPURatio:    types.Int64Value(data.ContainerCPURatio),
		ContainerMemoryRatio: types.Int64Value(data.ContainerMemoryRatio),
		Components:           ConvertRegionComponentsToModel(data.Components),
	}
}

func ConvertRegionComponentsToModel(data *entities.K8sClusterConfigRegionsComponents) *K8sClusterRegionComponentsModel {

	var result K8sClusterRegionComponentsModel
	if data == nil {
		return nil
	}

	val := reflect.ValueOf(*data)
	allFalse := true
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		switch field.Kind() {
		case reflect.Bool:
			if field.Bool() {
				allFalse = false
			}
		case reflect.Ptr:
			if !field.IsNil() {
				allFalse = false
			}
		}
	}
	if allFalse {
		return nil
	}

	dataVal := reflect.ValueOf(data)

	if dataVal.Kind() == reflect.Ptr {
		if dataVal.IsNil() {
			return nil
		}
		dataVal = dataVal.Elem()
	}
	dataType := dataVal.Type()

	resultVal := reflect.ValueOf(&result).Elem()

	for i := 0; i < dataVal.NumField(); i++ {
		fieldName := dataType.Field(i).Name

		resultField := resultVal.FieldByName(fieldName)
		if resultField.IsValid() && resultField.CanSet() && resultField.Type() == reflect.TypeOf(types.Bool{}) {
			fieldValue := dataVal.Field(i).Bool()
			if fieldValue {
				resultField.Set(reflect.ValueOf(types.BoolValue(fieldValue)))
			} else {
				resultField.Set(reflect.ValueOf(types.BoolNull()))
			}

		}
	}
	return &result
}

func ConvertIngressToModel(data entities.K8sClusterConfigIngress, isClusterIngress bool) K8sClusterIngressModel {
	var iscodes basetypes.ListValue
	if isClusterIngress {
		elements := make([]attr.Value, len(data.IsCodes))
		for i, code := range data.IsCodes {
			elements[i] = types.StringValue(code)
		}
		iscodes = basetypes.ListValue(types.SetValueMust(types.StringType, elements))
	} else {
		iscodes = types.ListNull(types.StringType)
	}

	return K8sClusterIngressModel{
		Name: types.StringValue(data.Name),
		Size: types.Int64Value(data.Size),
		Flavor: FlavorModel{
			Cores:  types.Int64Value(data.Flavor.Cpus),
			Memory: types.Int64Value(data.Flavor.Memory),
			Name:   types.StringValue(data.Flavor.Name),
			UUID:   types.StringValue(data.Flavor.UUID),
		},
		IsCodes:        iscodes,
		Features:       ConvertIngressFeaturesToModel(data.Features),
		TcpUdpSettings: ConvertTcpUdpToModel(data.TcpUdpSettings),
	}
}

func ConvertTcpUdpToModel(data []entities.K8sClusterConfigIngressTcpUdp) []K8sClusterIngressTcpUdpModel {
	var tcpUdpSettingsList []K8sClusterIngressTcpUdpModel
	for _, settings := range data {
		item := K8sClusterIngressTcpUdpModel{
			Mode:    types.StringValue(settings.Mode),
			Port:    types.Int64Value(settings.Port),
			Service: types.StringValue(settings.Service),
		}
		tcpUdpSettingsList = append(tcpUdpSettingsList, item)
	}
	return tcpUdpSettingsList
}

func ConvertIngressFeaturesToModel(data *entities.K8sClusterConfigIngressFeatures) *K8sClusterIngressFeaturesModel {
	if data == nil {
		return nil
	}

	return &K8sClusterIngressFeaturesModel{
		Http2Protocol: types.BoolValue(data.Http2Protocol),
		ProxyProtocol: types.BoolValue(data.ProxyProtocol),
	}
}

func ConvertComponentsToModel(ctx context.Context, data *entities.K8sClusterConfigComponents) *K8sClusterComponentsModel {
	if data == nil {
		return nil
	}

	return &K8sClusterComponentsModel{
		Istio:           ConvertIstioToModel(ctx, data.Istio),
		Astrom:          ConvertComponentToModel(data.Astrom),
		GpuOperator:     ConvertComponentToModel(data.GpuOperator),
		TridentOperator: ConvertComponentToModel(data.TridentOperator),
		TsamOperator:    ConvertComponentToModel(data.TsamOperator),
		TslgOperator:    ConvertComponentToModel(data.TslgOperator),
		TsdsOperator:    ConvertComponentToModel(data.TsdsOperator),
		ChaosMesh:       ConvertComponentToModel(data.ChaosMesh),
		Tyk:             ConvertComponentToModel(data.Tyk),
	}
}

func ConvertFeaturesToModel(data entities.K8sClusterConfigFeatures) K8sClusterCniPluginModel {

	return K8sClusterCniPluginModel{
		Name: types.StringValue(data.CniPlugin.Name),
	}
}

func ConvertIstioToModel(ctx context.Context, data *entities.K8sClusterConfigIstio) *K8sClusterIstioModel {

	if data == nil {
		return nil
	}
	options, _ := ConvertIstioOptionsToModel(ctx, data.Options)
	return &K8sClusterIstioModel{
		Options:       options,
		Installed:     types.BoolValue(data.Installed),
		ControlPlanes: ConvertIstioCPToModel(data.ControlPlanes),
		OptionsFlavor: types.StringValue(data.OptionsFlavor),
	}
}

func ConvertIstioOptionsToModel(ctx context.Context, data *entities.K8sClusterConfigIstioOptions) (types.Object, diag.Diagnostics) {
	if data == nil {
		return types.ObjectNull(map[string]attr.Type{
			"extauth_operator":     types.BoolType,
			"gateway_operator":     types.BoolType,
			"accesslogs_operator":  types.BoolType,
			"ratelimiter_operator": types.BoolType,
			"span_operator":        types.BoolType,
			"sm_operator":          types.BoolType,
		}), nil
	}

	return types.ObjectValue(
		map[string]attr.Type{
			"extauth_operator":     types.BoolType,
			"gateway_operator":     types.BoolType,
			"accesslogs_operator":  types.BoolType,
			"ratelimiter_operator": types.BoolType,
			"span_operator":        types.BoolType,
			"sm_operator":          types.BoolType,
		},
		map[string]attr.Value{
			"extauth_operator":     types.BoolPointerValue(data.ExtauthOperator),
			"gateway_operator":     types.BoolPointerValue(data.GatewayOperator),
			"accesslogs_operator":  types.BoolPointerValue(data.AccesslogsOperator),
			"ratelimiter_operator": types.BoolPointerValue(data.RatelimiterOperator),
			"span_operator":        types.BoolPointerValue(data.SpanOperator),
			"sm_operator":          types.BoolPointerValue(data.SmOperator),
		},
	)
}

func ConvertIstioCPToModel(data []entities.K8sClusterConfigIstioControlPlane) []K8sClusterIstioControlPlaneModel {
	var controlPlaneList []K8sClusterIstioControlPlaneModel
	for _, controlPlane := range data {
		cp := K8sClusterIstioControlPlaneModel{
			Name:   types.StringValue(controlPlane.Name),
			Flavor: types.StringValue(controlPlane.Flavor),
			Options: K8sClusterIstioCPOptionsModel{
				Eventrouter: types.BoolValue(controlPlane.Options.Eventrouter),
			},
		}
		controlPlaneList = append(controlPlaneList, cp)
	}
	return controlPlaneList
}

func ConvertControlPlaneToModel(data []entities.K8sClusterConfigControlPlane) []K8sClusterControlPlaneModel {
	var cpComponentList []K8sClusterControlPlaneModel
	for _, cpComponent := range data {
		var size types.Int64
		if cpComponent.RoleName == "infra" {
			size = types.Int64Value(cpComponent.Size)
		}

		comp := K8sClusterControlPlaneModel{
			Size: size,
			Flavor: FlavorModel{
				Cores:  types.Int64Value(cpComponent.Flavor.Cpus),
				Memory: types.Int64Value(cpComponent.Flavor.Memory),
				Name:   types.StringValue(cpComponent.Flavor.Name),
				UUID:   types.StringValue(cpComponent.Flavor.UUID),
			},
			Role: types.StringValue(cpComponent.RoleName),
		}
		cpComponentList = append(cpComponentList, comp)
	}
	return cpComponentList
}

func ConvertDefaultCPToModel(data []references.K8sClusterCPDefaults) []K8sClusterControlPlaneModel {
	var cpComponentList []K8sClusterControlPlaneModel
	for _, cpComponent := range data {

		comp := K8sClusterControlPlaneModel{
			Size: types.Int64Value(cpComponent.Size),
			Flavor: FlavorModel{
				Name: types.StringValue(cpComponent.Flavor),
			},
			Role: types.StringValue(cpComponent.Role),
		}
		cpComponentList = append(cpComponentList, comp)
	}
	return cpComponentList
}

func ConvertComponentToModel(data *entities.K8sClusterConfigFlag) *K8sClusterFlagModel {

	if data == nil {
		return nil
	} else {
		return &K8sClusterFlagModel{
			Installed: types.BoolValue(true),
		}
	}
}
