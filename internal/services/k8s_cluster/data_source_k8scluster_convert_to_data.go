package k8scluster

import (
	"context"
	"terraform-provider-vtb/pkg/client/entities"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func ConvertModelToRegion(model K8sClusterRegionModel) entities.K8sConfigRegion {

	var components entities.K8sConfigRegionComponents
	if model.Components != nil {
		components = *ConvertModelToRegionComponents(model.Components)
	}
	return entities.K8sConfigRegion{
		Name: model.Name.ValueString(),
		Size: model.Size.ValueInt64(),
		Flavor: entities.K8sConfigNodeFlavor{
			Cpus:   model.Flavor.Cores.ValueInt64(),
			Memory: model.Flavor.Memory.ValueInt64(),
			Name:   model.Flavor.Name.ValueString(),
			UUID:   model.Flavor.UUID.ValueString(),
		},
		IsCodes:              ConvertModelToIsCodes(model.IsCodes),
		PodCPUMax:            model.PodCPUMax.ValueInt64(),
		PodMemoryMax:         model.PodMemoryMax.ValueInt64(),
		ContainerCPURatio:    model.ContainerCPURatio.ValueInt64(),
		ContainerMemoryRatio: model.ContainerMemoryRatio.ValueInt64(),
		Components:           &components,
	}
}

func ConvertModelToIngress(model K8sClusterIngressModel) entities.K8sConfigIngress {

	return entities.K8sConfigIngress{
		Name: model.Name.ValueString(),
		Size: model.Size.ValueInt64(),
		Flavor: entities.K8sConfigNodeFlavor{
			Cpus:   model.Flavor.Cores.ValueInt64(),
			Memory: model.Flavor.Memory.ValueInt64(),
			Name:   model.Flavor.Name.ValueString(),
			UUID:   model.Flavor.UUID.ValueString(),
		},
		IsCodes:        ConvertModelToIsCodes(model.IsCodes),
		Features:       ConvertModelToIngressFeatures(model.Features),
		TcpUdpSettings: ConvertModelToTcpUdpSettings(model.TcpUdpSettings),
	}
}

func ConvertModelToTcpUdpSettings(model []K8sClusterIngressTcpUdpModel) []entities.K8sConfigIngressTcpUdp {
	var tcpUdpSettingsList []entities.K8sConfigIngressTcpUdp
	for _, settings := range model {
		item := entities.K8sConfigIngressTcpUdp{
			Mode:    settings.Mode.ValueString(),
			Port:    settings.Port.ValueInt64(),
			Service: settings.Service.ValueString(),
		}
		tcpUdpSettingsList = append(tcpUdpSettingsList, item)
	}
	return tcpUdpSettingsList
}

func ConvertModelToIngressFeatures(model *K8sClusterIngressFeaturesModel) *entities.K8sConfigIngressFeatures {
	if model == nil {
		return nil
	}

	return &entities.K8sConfigIngressFeatures{
		Http2Protocol: model.Http2Protocol.ValueBool(),
		ProxyProtocol: model.ProxyProtocol.ValueBool(),
	}
}

func ConvertModelToFeatures(ctx context.Context, model K8sClusterModel) (entities.K8sClusterConfigFeatures, diag.Diagnostics) {
	var diags diag.Diagnostics
	istioFlag := model.Components != nil &&
		model.Components.Istio != nil &&
		!model.Components.Istio.Installed.IsNull() &&
		model.Components.Istio.Installed.ValueBool()

	var istioOptions entities.K8sClusterConfigIstioOptions
	var istioOptionsDiags diag.Diagnostics
	if model.Components != nil &&
		model.Components.Istio != nil &&
		!model.Components.Istio.Options.IsNull() {
		istioOptions, istioOptionsDiags = ConvertModelToIstioOptions(ctx, model.Components.Istio.Options)
		diags.Append(istioOptionsDiags...)
		if istioOptionsDiags.HasError() {
			return entities.K8sClusterConfigFeatures{}, diags
		}
	} else {
		istioOptions = entities.K8sClusterConfigIstioOptions{}
	}

	return entities.K8sClusterConfigFeatures{
		Console:    true,
		Monitoring: true,
		CniPlugin: entities.K8sClusterConfigCniPlugin{
			Name: model.CniPlugin.Name.ValueString(),
		},
		Istio:        istioFlag,
		IstioOptions: &istioOptions,
	}, diags
}

func ConvertModelToIstio(ctx context.Context, model *K8sClusterIstioModel) (entities.K8sClusterConfigIstio, diag.Diagnostics) {
	var diags diag.Diagnostics
	var cpList []entities.K8sClusterConfigIstioControlPlane
	for _, cp := range model.ControlPlanes {
		cpList = append(cpList, ConvertModelToIstioCP(cp))
	}
	options, optionsDiags := ConvertModelToIstioOptions(ctx, model.Options)
	diags.Append(optionsDiags...)
	if optionsDiags.HasError() {
		return entities.K8sClusterConfigIstio{}, diags
	}
	return entities.K8sClusterConfigIstio{
		Options:       &options,
		Installed:     model.Installed.ValueBool(),
		ControlPlanes: cpList,
		OptionsFlavor: model.OptionsFlavor.ValueString(),
	}, diags
}

func ConvertModelToIstioOptions(ctx context.Context, model types.Object) (entities.K8sClusterConfigIstioOptions, diag.Diagnostics) {
	jsonData := entities.K8sClusterConfigIstioOptions{}
	var diags diag.Diagnostics

	var tfModel K8sClusterIstioOptionsModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return jsonData, diags
	}

	jsonData.ExtauthOperator = tfModel.ExtauthOperator.ValueBoolPointer()
	jsonData.GatewayOperator = tfModel.GatewayOperator.ValueBoolPointer()
	jsonData.AccesslogsOperator = tfModel.AccesslogsOperator.ValueBoolPointer()
	jsonData.RatelimiterOperator = tfModel.RatelimiterOperator.ValueBoolPointer()
	jsonData.SmOperator = tfModel.SmOperator.ValueBoolPointer()
	jsonData.SpanOperator = tfModel.SpanOperator.ValueBoolPointer()

	return jsonData, diags
}

func ConvertModelToIstioCP(model K8sClusterIstioControlPlaneModel) entities.K8sClusterConfigIstioControlPlane {

	return entities.K8sClusterConfigIstioControlPlane{
		Name:    model.Name.ValueString(),
		Flavor:  model.Flavor.ValueString(),
		Options: ConvertModelToIstioCPOptions(&model.Options),
	}
}

func ConvertModelToIstioCPOptions(model *K8sClusterIstioCPOptionsModel) *entities.K8sClusterConfigIstioCPOptions {
	if model == nil {
		return nil
	}
	return &entities.K8sClusterConfigIstioCPOptions{
		Eventrouter: model.Eventrouter.ValueBool(),
	}
}

func ConvertModelToLimits(model K8sClusterModel) entities.Limits {

	return entities.Limits{
		ContainerCPURatio:    model.ContainerCPURatio.ValueInt64(),
		ContainerMemoryRatio: model.ContainerMemoryRatio.ValueInt64(),
	}
}

func ConvertModelToCPComponent(model K8sClusterControlPlaneModel) entities.K8sClusterConfigControlPlane {
	var size int64
	if model.Role.ValueString() == "infra" {
		size = model.Size.ValueInt64()
	}
	return entities.K8sClusterConfigControlPlane{
		RoleName: model.Role.ValueString(),
		Size:     size,
		Flavor: entities.K8sConfigNodeFlavor{
			Cpus:   model.Flavor.Cores.ValueInt64(),
			Memory: model.Flavor.Memory.ValueInt64(),
			Name:   model.Flavor.Name.ValueString(),
			UUID:   model.Flavor.UUID.ValueString(),
		},
	}
}
func ConvertModelToRegionComponents(model *K8sClusterRegionComponentsModel) *entities.K8sConfigRegionComponents {
	if model == nil {
		return nil
	}
	return &entities.K8sConfigRegionComponents{
		Astrom:       model.Astrom.ValueBool(),
		TsamOperator: model.TsamOperator.ValueBool(),
		ChaosMesh:    model.ChaosMesh.ValueBool(),
		TsdsOperator: model.TsdsOperator.ValueBool(),
		TslgOperator: model.TslgOperator.ValueBool(),
		Tyk:          model.Tyk.ValueBool(),
	}
}

func ConvertModelToIsCodes(list types.List) []string {
	isCodes := make([]string, 0, len(list.Elements()))
	for _, elem := range list.Elements() {
		isCodes = append(isCodes, elem.(basetypes.StringValue).ValueString())
	}
	return isCodes
}

func ConvertModelToVersion(model *K8sClusterVersionModel) entities.K8sClusterConfig {
	return entities.K8sClusterConfig{
		Version:        model.K8sVersion.ValueString(),
		ProductVersion: model.ProductVersion.ValueString(),
	}
}
