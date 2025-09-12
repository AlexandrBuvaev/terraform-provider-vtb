package provider

import (
	"context"
	"terraform-provider-vtb/pkg/client/entities"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ConvertModelToConfig converts a ConfigModel to a Config struct.
func ConvertModelToConfig(ctx context.Context, model types.Object) (*entities.Config, diag.Diagnostics) {
	jsonData := &entities.Config{}
	var diags diag.Diagnostics

	var tfModel BalancerV3ConfigDataSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var ports []entities.Port
	if !tfModel.Ports.IsNull() && !tfModel.Ports.IsUnknown() {
		var portObjects []types.Object
		diags.Append(tfModel.Ports.ElementsAs(ctx, &portObjects, false)...)
		if !diags.HasError() {
			ports = make([]entities.Port, len(portObjects))
			for i, portObj := range portObjects {
				port, portDiags := ConvertModelToPort(ctx, portObj)
				diags.Append(portDiags...)
				if portDiags.HasError() {
					return nil, diags
				}
				ports[i] = *port
			}
		}
	}

	var backends []entities.Backend
	if !tfModel.Backends.IsNull() && !tfModel.Backends.IsUnknown() {
		var backendObjects []types.Object
		diags.Append(tfModel.Backends.ElementsAs(ctx, &backendObjects, false)...)
		if !diags.HasError() {
			backends = make([]entities.Backend, len(backendObjects))
			for i, backendObj := range backendObjects {
				backend, backendDiags := ConvertModelToBackend(ctx, backendObj)
				diags.Append(backendDiags...)
				if backendDiags.HasError() {
					return nil, diags
				}
				backends[i] = *backend
			}
		}
	}

	var publications []entities.Publication
	if !tfModel.Publications.IsNull() && !tfModel.Publications.IsUnknown() {
		var publicationObjects []types.Object
		diags.Append(tfModel.Publications.ElementsAs(ctx, &publicationObjects, false)...)
		if !diags.HasError() {
			publications = make([]entities.Publication, len(publicationObjects))
			for i, publicationObj := range publicationObjects {
				publication, pubDiags := ConvertModelToPublication(ctx, publicationObj)
				diags.Append(pubDiags...)
				if pubDiags.HasError() {
					return nil, diags
				}
				publications[i] = *publication
			}
		}
	}

	var globals entities.Globals
	if !tfModel.Globals.IsNull() && !tfModel.Globals.IsUnknown() {
		var globalsDiags diag.Diagnostics
		globals, globalsDiags = ConvertModelToGlobals(ctx, tfModel.Globals)
		diags.Append(globalsDiags...)
		if globalsDiags.HasError() {
			return nil, diags
		}
	}

	var defaults entities.Defaults
	if !tfModel.Defaults.IsNull() && !tfModel.Defaults.IsUnknown() {
		var defaultsDiags diag.Diagnostics
		defaults, defaultsDiags = ConvertModelToDefaults(ctx, tfModel.Defaults)
		diags.Append(defaultsDiags...)
		if defaultsDiags.HasError() {
			return nil, diags
		}
	}

	jsonData.Ports = ports
	jsonData.Backends = backends
	jsonData.Publications = publications
	jsonData.Globals = globals
	jsonData.Defaults = defaults

	return jsonData, diags
}

// ConvertModelToPort converts a PortModel to a Port struct.
func ConvertModelToPort(ctx context.Context, model types.Object) (*entities.Port, diag.Diagnostics) {
	jsonData := &entities.Port{}
	var diags diag.Diagnostics

	var tfModel BalancerV3PortSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var keepAlive *entities.PortKeepAlive
	if !tfModel.KeepAlive.IsNull() && !tfModel.KeepAlive.IsUnknown() {
		var keepAliveDiags diag.Diagnostics
		keepAlive, keepAliveDiags = ConvertModelToPortKeepAlive(ctx, tfModel.KeepAlive)
		diags.Append(keepAliveDiags...)
		if keepAliveDiags.HasError() {
			return nil, diags
		}
	}

	var redirect *entities.Redirect
	if !tfModel.Redirect.IsNull() && !tfModel.Redirect.IsUnknown() {
		var redirectDiags diag.Diagnostics
		redirect, redirectDiags = ConvertModelToRedirect(ctx, tfModel.Redirect)
		diags.Append(redirectDiags...)
		if redirectDiags.HasError() {
			return nil, diags
		}
	}

	var prescription *entities.Prescription
	if !tfModel.Prescription.IsNull() && !tfModel.Prescription.IsUnknown() {
		var prescriptionDiags diag.Diagnostics
		prescription, prescriptionDiags = ConvertModelToPrescription(ctx, tfModel.Prescription)
		diags.Append(prescriptionDiags...)
		if prescriptionDiags.HasError() {
			return nil, diags
		}
	}

	var httpSettings *entities.HttpSettings
	if !tfModel.HttpSettings.IsNull() && !tfModel.HttpSettings.IsUnknown() {
		var httpSettingsDiags diag.Diagnostics
		httpSettings, httpSettingsDiags = ConvertModelToHttpSettings(ctx, tfModel.HttpSettings)
		diags.Append(httpSettingsDiags...)
		if httpSettingsDiags.HasError() {
			return nil, diags
		}
	}

	jsonData.Mode = tfModel.Mode.ValueString()
	jsonData.Port = tfModel.Port.ValueInt64()
	jsonData.Maxconn = tfModel.Maxconn.ValueInt64()
	jsonData.TlsProfile = tfModel.TlsProfile.ValueStringPointer()

	jsonData.KeepAlive = keepAlive
	jsonData.Redirect = redirect
	jsonData.Prescription = prescription
	jsonData.HttpSettings = httpSettings

	return jsonData, diags
}

// ConvertModelToPortKeepAlive converts a PortKeepAliveModel to a PortKeepAlive struct.
func ConvertModelToPortKeepAlive(ctx context.Context, model types.Object) (*entities.PortKeepAlive, diag.Diagnostics) {
	jsonData := &entities.PortKeepAlive{}
	var diags diag.Diagnostics

	var tfModel PortKeepAliveModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var tcp *entities.TCPKeepAlive
	if !tfModel.Tcp.IsNull() && !tfModel.Tcp.IsUnknown() {
		var tcpDiags diag.Diagnostics
		tcp, tcpDiags = ConvertModelToTCPKeepAlive(ctx, tfModel.Tcp)
		diags.Append(tcpDiags...)
		if tcpDiags.HasError() {
			return nil, diags
		}
	}

	var http *entities.HTTPKeepAlive
	if !tfModel.Http.IsNull() && !tfModel.Http.IsUnknown() {
		var httpDiags diag.Diagnostics
		http, httpDiags = ConvertModelToHTTPKeepAlive(ctx, tfModel.Http)
		diags.Append(httpDiags...)
		if httpDiags.HasError() {
			return nil, diags
		}
	}

	jsonData.Tcp = tcp
	jsonData.Http = http

	return jsonData, diags
}

// ConvertModelToTCPKeepAlive converts a TCPKeepAliveModel to a TCPKeepAlive struct.
func ConvertModelToTCPKeepAlive(ctx context.Context, model types.Object) (*entities.TCPKeepAlive, diag.Diagnostics) {
	jsonData := &entities.TCPKeepAlive{}
	var diags diag.Diagnostics

	var tfModel TCPKeepAliveModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData.Mode = tfModel.Mode.ValueStringPointer()
	jsonData.Count = tfModel.Count.ValueInt64Pointer()
	jsonData.Timer = tfModel.Timer.ValueInt64Pointer()

	return jsonData, diags
}

// ConvertModelToHTTPKeepAlive converts a HTTPKeepAliveModel to a HTTPKeepAlive struct.
func ConvertModelToHTTPKeepAlive(ctx context.Context, model types.Object) (*entities.HTTPKeepAlive, diag.Diagnostics) {
	jsonData := &entities.HTTPKeepAlive{}
	var diags diag.Diagnostics

	var tfModel HTTPKeepAliveModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData.Mode = tfModel.Mode.ValueStringPointer()
	jsonData.Timer = tfModel.Timer.ValueInt64Pointer()

	return jsonData, diags
}

// ConvertModelToRedirect converts a RedirectModel to a Redirect struct.
func ConvertModelToRedirect(ctx context.Context, model types.Object) (*entities.Redirect, diag.Diagnostics) {
	jsonData := &entities.Redirect{}
	var diags diag.Diagnostics

	var tfModel RedirectModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData.Enabled = tfModel.Enabled.ValueBool()

	return jsonData, diags
}

// ConvertModelToPrescription converts a PrescriptionModel to a Prescription struct.
func ConvertModelToPrescription(ctx context.Context, model types.Object) (*entities.Prescription, diag.Diagnostics) {
	jsonData := &entities.Prescription{}
	var diags diag.Diagnostics

	var tfModel PrescriptionModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData.Enabled = tfModel.Enabled.ValueBool()
	jsonData.LiveTime = tfModel.LiveTime.ValueInt64Pointer()
	jsonData.IncludeSubdomains = tfModel.IncludeSubdomains.ValueBoolPointer()

	return jsonData, diags
}

func ConvertModelToVersionActivation(ctx context.Context, model types.Object) (*entities.VersionActivation, diag.Diagnostics) {
	jsonData := &entities.VersionActivation{}
	var diags diag.Diagnostics

	var tfModel VersionActivationSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData.Version = tfModel.Version.ValueString()

	return jsonData, diags
}

// ConvertModelToHttpSettings converts a HttpSettingsModel to a HttpSettings struct.
func ConvertModelToHttpSettings(ctx context.Context, model types.Object) (*entities.HttpSettings, diag.Diagnostics) {
	var diags diag.Diagnostics
	jsonData := &entities.HttpSettings{}

	var tfModel HttpSettingsSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	// Convert VersionActivationPriority if present
	if !tfModel.VersionActivationPriority.IsNull() && !tfModel.VersionActivationPriority.IsUnknown() {
		var versionObjects []types.Object
		diags.Append(tfModel.VersionActivationPriority.ElementsAs(ctx, &versionObjects, false)...)
		if diags.HasError() {
			return nil, diags
		}

		versions := make([]*entities.VersionActivation, len(versionObjects))
		for i, versionObj := range versionObjects {
			version, versionDiags := ConvertModelToVersionActivation(ctx, versionObj)
			diags.Append(versionDiags...)
			if versionDiags.HasError() {
				return nil, diags
			}
			versions[i] = version
		}
	}

	// Set other fields
	jsonData.GrpcOverHttp2 = tfModel.GrpcOverHttp2.ValueBool()

	return jsonData, diags
}

// ConvertModelToGlobals converts a GlobalsModel to a Globals struct.
func ConvertModelToGlobals(ctx context.Context, model types.Object) (entities.Globals, diag.Diagnostics) {
	jsonData := entities.Globals{}
	var diags diag.Diagnostics

	var tfModel BalancerV3GlobalsSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return jsonData, diags
	}

	jsonData.Maxconn = tfModel.Maxconn.ValueInt64Pointer()
	jsonData.TuneOptions = tfModel.TuneOptions.ValueStringPointer()
	jsonData.H2WorkaroundBogusWebsocketClients = tfModel.H2WorkaroundBogusWebsocketClients.ValueBool()

	return jsonData, diags
}

// ConvertModelToDefaults converts a DefaultsModel to a Defaults struct.
func ConvertModelToDefaults(ctx context.Context, model types.Object) (entities.Defaults, diag.Diagnostics) {
	jsonData := entities.Defaults{}
	var diags diag.Diagnostics

	var tfModel BalancerV3DefaultsSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return jsonData, diags
	}

	jsonData.ClientTimeout = tfModel.ClientTimeout.ValueInt64()
	jsonData.ServerTimeout = tfModel.ServerTimeout.ValueInt64()
	jsonData.ConnectTimeout = tfModel.ConnectTimeout.ValueInt64()

	return jsonData, diags
}

// ConvertModelToBackend converts a BackendModel to a Backend struct.
func ConvertModelToBackend(ctx context.Context, model types.Object) (*entities.Backend, diag.Diagnostics) {
	jsonData := &entities.Backend{}
	var diags diag.Diagnostics

	var tfModel BalancerV3BackendSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var retries *entities.Retries
	if !tfModel.Retries.IsNull() && !tfModel.Retries.IsUnknown() {
		var retriesDiags diag.Diagnostics
		retries, retriesDiags = ConvertModelToRetries(ctx, tfModel.Retries)
		diags.Append(retriesDiags...)
		if retriesDiags.HasError() {
			return nil, diags
		}
	}

	var servers []*entities.Server
	if !tfModel.Servers.IsNull() && !tfModel.Servers.IsUnknown() {
		var serversDiags diag.Diagnostics
		servers, serversDiags = ConvertModelToServers(ctx, tfModel.Servers)
		diags.Append(serversDiags...)
		if serversDiags.HasError() {
			return nil, diags
		}
	}

	var healthcheck *entities.Healthcheck
	if !tfModel.Healthcheck.IsNull() && !tfModel.Healthcheck.IsUnknown() {
		var healthcheckDiags diag.Diagnostics
		healthcheck, healthcheckDiags = ConvertModelToHealthcheck(ctx, tfModel.Healthcheck)
		diags.Append(healthcheckDiags...)
		if healthcheckDiags.HasError() {
			return nil, diags
		}
	}

	var cookie *entities.Cookie
	if !tfModel.Cookie.IsNull() && !tfModel.Cookie.IsUnknown() {
		var cookieDiags diag.Diagnostics
		cookie, cookieDiags = ConvertModelToCookie(ctx, tfModel.Cookie)
		diags.Append(cookieDiags...)
		if cookieDiags.HasError() {
			return nil, diags
		}
	}

	var forwardfor *entities.Forwardfor
	if !tfModel.Forwardfor.IsNull() && !tfModel.Forwardfor.IsUnknown() {
		var forwardforDiags diag.Diagnostics
		forwardfor, forwardforDiags = ConvertModelToForwardfor(ctx, tfModel.Forwardfor)
		diags.Append(forwardforDiags...)
		if forwardforDiags.HasError() {
			return nil, diags
		}
	}

	var httpReuse *entities.HttpReuse
	if !tfModel.HttpReuse.IsNull() && !tfModel.HttpReuse.IsUnknown() {
		var httpReuseDiags diag.Diagnostics
		httpReuse, httpReuseDiags = ConvertModelToHttpReuse(ctx, tfModel.HttpReuse)
		diags.Append(httpReuseDiags...)
		if httpReuseDiags.HasError() {
			return nil, diags
		}
	}

	var keepAlive *entities.BackendKeepAlive
	if !tfModel.KeepAlive.IsNull() && !tfModel.KeepAlive.IsUnknown() {
		var keepAliveDiags diag.Diagnostics
		keepAlive, keepAliveDiags = ConvertModelToBackendKeepAlive(ctx, tfModel.KeepAlive)
		diags.Append(keepAliveDiags...)
		if keepAliveDiags.HasError() {
			return nil, diags
		}
	}

	var serversSettings *entities.ServerSettings
	if !tfModel.ServersSettings.IsNull() && !tfModel.ServersSettings.IsUnknown() {
		var serversSettingsDiags diag.Diagnostics
		serversSettings, serversSettingsDiags = ConvertModelToServerSettings(ctx, tfModel.ServersSettings)
		diags.Append(serversSettingsDiags...)
		if serversSettingsDiags.HasError() {
			return nil, diags
		}
	}

	jsonData.Mode = tfModel.Mode.ValueString()
	jsonData.Globalname = tfModel.Globalname.ValueString()
	jsonData.BackendName = tfModel.BackendName.ValueString()
	jsonData.BalancingAlgorithm = tfModel.BalancingAlgorithm.ValueString()

	jsonData.Retries = retries
	jsonData.Servers = servers
	jsonData.Healthcheck = healthcheck
	jsonData.Cookie = cookie
	jsonData.Forwardfor = forwardfor
	jsonData.HttpReuse = httpReuse
	jsonData.KeepAlive = keepAlive
	jsonData.ServersSettings = serversSettings

	return jsonData, diags
}

// ConvertModelToRetries converts a RetriesModel to a Retries struct.
func ConvertModelToRetries(ctx context.Context, model types.Object) (*entities.Retries, diag.Diagnostics) {
	jsonData := &entities.Retries{}
	var diags diag.Diagnostics

	var tfModel RetriesSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var conditions []string
	if !tfModel.Conditions.IsNull() && !tfModel.Conditions.IsUnknown() {
		var conditionList []basetypes.StringValue
		diags.Append(tfModel.Conditions.ElementsAs(ctx, &conditionList, false)...)
		if !diags.HasError() {
			conditions = make([]string, len(conditionList))
			for i, condition := range conditionList {
				conditions[i] = condition.ValueString()
			}
		}
	}

	jsonData.Enabled = tfModel.Enabled.ValueBool()
	jsonData.Count = tfModel.Count.ValueInt64()
	jsonData.Conditions = conditions
	jsonData.Redispatch = tfModel.Redispatch.ValueStringPointer()

	return jsonData, diags
}

// ConvertModelToServers converts a slice of ServerModel to a slice of Server struct.
func ConvertModelToServers(ctx context.Context, models types.List) ([]*entities.Server, diag.Diagnostics) {
	var diags diag.Diagnostics
	jsonData := make([]*entities.Server, 0)

	if models.IsNull() || models.IsUnknown() {
		return jsonData, diags
	}

	// Конвертируем элементы списка
	var serverModels []ServerSourceModel
	diags.Append(models.ElementsAs(ctx, &serverModels, false)...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData = make([]*entities.Server, len(serverModels))
	for i, model := range serverModels {
		jsonData[i] = &entities.Server{
			Name:      model.Name.ValueString(),
			State:     model.State.ValueString(),
			Address:   model.Address.ValueString(),
			SendProxy: model.SendProxy.ValueStringPointer(),
			Maxconn:   model.Maxconn.ValueInt64(),
			Weight:    model.Weight.ValueInt64(),
		}
	}

	return jsonData, diags
}

// ConvertModelToHealthcheck converts a HealthcheckModel to a Healthcheck struct.
func ConvertModelToHealthcheck(ctx context.Context, model types.Object) (*entities.Healthcheck, diag.Diagnostics) {
	jsonData := &entities.Healthcheck{}
	var diags diag.Diagnostics

	var tfModel HealthcheckSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var checkStrings []*entities.CheckString
	if !tfModel.CheckStrings.IsNull() && !tfModel.CheckStrings.IsUnknown() {
		var checkStringObjects []types.Object
		diags.Append(tfModel.CheckStrings.ElementsAs(ctx, &checkStringObjects, false)...)
		if !diags.HasError() {
			checkStrings = make([]*entities.CheckString, len(checkStringObjects))
			for i, checkStringObj := range checkStringObjects {
				checkString, checkDiags := ConvertModelToCheckString(ctx, checkStringObj)
				diags.Append(checkDiags...)
				if checkDiags.HasError() {
					return nil, diags
				}
				checkStrings[i] = checkString
			}
		}
	}

	jsonData.Mode = tfModel.Mode.ValueString()
	jsonData.Interval = tfModel.Interval.ValueInt64Pointer()
	jsonData.FallCount = tfModel.FallCount.ValueInt64Pointer()
	jsonData.RiseCount = tfModel.RiseCount.ValueInt64Pointer()
	jsonData.CheckStrings = checkStrings

	return jsonData, diags
}

// ConvertModelToCheckString converts a CheckStringModel to a CheckString struct.
func ConvertModelToCheckString(ctx context.Context, model types.Object) (*entities.CheckString, diag.Diagnostics) {
	jsonData := &entities.CheckString{}
	var diags diag.Diagnostics

	var tfModel CheckStringSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData.Uri = tfModel.Uri.ValueStringPointer()
	jsonData.Port = tfModel.Port.ValueInt64Pointer()
	jsonData.Match = tfModel.Match.ValueStringPointer()
	jsonData.Value = tfModel.Value.ValueStringPointer()
	jsonData.Method = tfModel.Method.ValueStringPointer()
	jsonData.Address = tfModel.Address.ValueStringPointer()
	jsonData.UseSsl = tfModel.UseSsl.ValueStringPointer()
	jsonData.Version = tfModel.Version.ValueStringPointer()
	jsonData.HostHdr = tfModel.HostHdr.ValueStringPointer()
	jsonData.SendProxy = tfModel.SendProxy.ValueStringPointer()
	jsonData.Body = tfModel.Body.ValueStringPointer()

	return jsonData, diags
}

// ConvertModelToCookie converts a CookieModel to a Cookie struct.
func ConvertModelToCookie(ctx context.Context, model types.Object) (*entities.Cookie, diag.Diagnostics) {
	jsonData := &entities.Cookie{}
	var diags diag.Diagnostics

	var tfModel CookieSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData.Enable = tfModel.Enable.ValueBool()
	jsonData.Secure = tfModel.Secure.ValueBoolPointer()

	return jsonData, diags
}

// ConvertModelToForwardfor converts a ForwardforModel to a Forwardfor struct.
func ConvertModelToForwardfor(ctx context.Context, model types.Object) (*entities.Forwardfor, diag.Diagnostics) {
	jsonData := &entities.Forwardfor{}
	var diags diag.Diagnostics

	var tfModel ForwardforSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData.Xff = tfModel.Xff.ValueBool()
	jsonData.Xcip = tfModel.Xcip.ValueBool()
	jsonData.Xrip = tfModel.Xrip.ValueBool()

	return jsonData, diags
}

// ConvertModelToHttpReuse converts a HttpReuseModel to a HttpReuse struct.
func ConvertModelToHttpReuse(ctx context.Context, model types.Object) (*entities.HttpReuse, diag.Diagnostics) {
	jsonData := &entities.HttpReuse{}
	var diags diag.Diagnostics

	var tfModel HttpReuseSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	jsonData.Mode = tfModel.Mode.ValueStringPointer()
	jsonData.PoolLowConn = tfModel.PoolLowConn.ValueInt64Pointer()
	jsonData.PoolMaxConn = tfModel.PoolMaxConn.ValueInt64Pointer()
	jsonData.PoolPurgeDelay = tfModel.PoolPurgeDelay.ValueInt64Pointer()

	return jsonData, diags
}

// ConvertModelToBackendKeepAlive converts a BackendKeepAliveModel to a BackendKeepAlive struct.
func ConvertModelToBackendKeepAlive(ctx context.Context, model types.Object) (*entities.BackendKeepAlive, diag.Diagnostics) {
	jsonData := &entities.BackendKeepAlive{}
	var diags diag.Diagnostics

	var tfModel BackendKeepAliveSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	jsonData.Mode = tfModel.Mode.ValueStringPointer()
	jsonData.Timer = tfModel.Timer.ValueInt64Pointer()

	return jsonData, diags
}

// ConvertModelToServerSettings converts a ServerSettingsModel to a ServerSettings struct.
func ConvertModelToServerSettings(ctx context.Context, model types.Object) (*entities.ServerSettings, diag.Diagnostics) {
	var diags diag.Diagnostics
	jsonData := &entities.ServerSettings{}

	var tfModel ServerSettingsSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var httpSettings *entities.HttpSettings
	if !tfModel.HttpSettings.IsNull() && !tfModel.HttpSettings.IsUnknown() {
		var httpSettingsDiag diag.Diagnostics
		httpSettings, httpSettingsDiag = ConvertModelToHttpSettings(ctx, tfModel.HttpSettings)
		diags.Append(httpSettingsDiag...)
		if httpSettingsDiag.HasError() {
			return nil, diags
		}
	}

	jsonData.Port = tfModel.Port.ValueInt64()
	jsonData.SlowStart = tfModel.SlowStart.ValueInt64Pointer()
	jsonData.TlsProfile = tfModel.TlsProfile.ValueStringPointer()
	jsonData.HttpSettings = httpSettings
	jsonData.UseSni = tfModel.UseSni.ValueBoolPointer()

	return jsonData, diags
}

// ConvertModelToPublication converts a PublicationModel to a Publication struct.
func ConvertModelToPublication(ctx context.Context, model types.Object) (*entities.Publication, diag.Diagnostics) {
	var diags diag.Diagnostics
	jsonData := &entities.Publication{}

	var tfModel BalancerV3PublicationSourceModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var routesObjects []types.Object
	jsonData.Routes = make([]entities.Route, len(routesObjects))
	if !tfModel.Routes.IsNull() && !tfModel.Routes.IsUnknown() {
		diags.Append(tfModel.Routes.ElementsAs(ctx, &routesObjects, false)...)
		if diags.HasError() {
			return nil, diags
		}
		jsonData.Routes = make([]entities.Route, len(routesObjects))
		for i, obj := range routesObjects {
			route, routeDiags := ConvertModelToRoute(ctx, obj)
			diags.Append(routeDiags...)
			if routeDiags.HasError() {
				return nil, diags
			}
			jsonData.Routes[i] = *route
		}
	}

	standinBackends := []string{}
	if !tfModel.StandinBackends.IsNull() && !tfModel.StandinBackends.IsUnknown() {
		diags.Append(tfModel.StandinBackends.ElementsAs(ctx, &standinBackends, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}

	var cnames []string
	if !tfModel.Cnames.IsNull() && !tfModel.Cnames.IsUnknown() {
		diags.Append(tfModel.Cnames.ElementsAs(ctx, &cnames, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}
	jsonData.Cnames = cnames
	jsonData.StandinBackends = standinBackends
	jsonData.Mode = tfModel.Mode.ValueString()
	jsonData.Port = tfModel.Port.ValueInt64()
	jsonData.Globalname = tfModel.Globalname.ValueString()
	jsonData.MainBackend = tfModel.MainBackend.ValueString()
	jsonData.AliveServCount = tfModel.AliveServCount.ValueInt64()
	jsonData.DefaultRouting = tfModel.DefaultRouting.ValueBool()

	return jsonData, diags
}

// ConvertModelToRoute converts a RouteModel to a Route struct.
func ConvertModelToRoute(ctx context.Context, model types.Object) (*entities.Route, diag.Diagnostics) {
	var diags diag.Diagnostics
	jsonData := &entities.Route{}

	var tfModel RouteModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	if !tfModel.Uri.IsNull() && !tfModel.Uri.IsUnknown() {
		var uriObjects []types.Object
		diags.Append(tfModel.Uri.ElementsAs(ctx, &uriObjects, false)...)

		jsonData.Uri = make([]entities.Endpoint, len(uriObjects))
		for i, obj := range uriObjects {
			endpoint, endpointDiags := ConvertModelToEndpoint(ctx, obj)
			diags.Append(endpointDiags...)
			jsonData.Uri[i] = *endpoint
		}
	}

	if !tfModel.Headers.IsNull() && !tfModel.Headers.IsUnknown() {
		var headerObjects []types.Object
		diags.Append(tfModel.Headers.ElementsAs(ctx, &headerObjects, false)...)

		jsonData.Headers = make([]entities.Header, len(headerObjects))
		for i, obj := range headerObjects {
			header, headerDiags := ConvertModelToHeader(ctx, obj)
			diags.Append(headerDiags...)
			jsonData.Headers[i] = *header
		}
	}

	if !tfModel.Ips.IsNull() && !tfModel.Ips.IsUnknown() {
		var ips []string
		diags.Append(tfModel.Ips.ElementsAs(ctx, &ips, false)...)
		jsonData.Ips = ips
	}

	if diags.HasError() {
		return nil, diags
	}
	return jsonData, diags
}

// ConvertModelToEndpoint converts an EndpointModel to an Endpoint struct.
func ConvertModelToEndpoint(ctx context.Context, model types.Object) (*entities.Endpoint, diag.Diagnostics) {
	var diags diag.Diagnostics
	var tfModel EndpointModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	condition := tfModel.Condition.ValueString()

	endpoints := make([]string, 0, len(tfModel.Endpoints.Elements()))
	diags.Append(tfModel.Endpoints.ElementsAs(ctx, &endpoints, false)...)
	if diags.HasError() {
		return nil, diags
	}

	return &entities.Endpoint{
		Condition: condition,
		Endpoints: endpoints,
	}, diags
}

// ConvertModelToHeader converts a HeaderModel to a Header struct.
func ConvertModelToHeader(ctx context.Context, model types.Object) (*entities.Header, diag.Diagnostics) {
	var diags diag.Diagnostics
	var tfModel HeaderModel
	diags.Append(model.As(ctx, &tfModel, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	return &entities.Header{
		Name:      tfModel.Name.ValueString(),
		CheckType: tfModel.CheckType.ValueString(),
		Value:     tfModel.Value.ValueStringPointer(),
	}, diags
}
