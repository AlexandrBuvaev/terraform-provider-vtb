package provider

import (
	"reflect"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Use in backend and port
type HttpSettingsSourceModel struct {
	GrpcOverHttp2             types.Bool `tfsdk:"grpc_over_http2"`
	VersionActivationPriority types.List `tfsdk:"version_activation_priority"`
}

func (m HttpSettingsSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"grpc_over_http2":             types.BoolType,
		"version_activation_priority": types.ListType{ElemType: types.ObjectType{AttrTypes: VersionActivationSourceModel{}.AttrTypes()}},
	}
}

type VersionActivationSourceModel struct {
	Version types.String `tfsdk:"version"`
}

func (m VersionActivationSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"version": types.StringType,
	}
}

var BalancerV3HttpSettingsScheme = schema.SingleNestedAttribute{
	MarkdownDescription: "HTTP settings",
	Optional:            true,
	Attributes: map[string]schema.Attribute{
		"grpc_over_http2": schema.BoolAttribute{
			MarkdownDescription: "Use gRPC over HTTP2",
			Optional:            true,
		},
		"version_activation_priority": schema.ListNestedAttribute{
			MarkdownDescription: "List of versions",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"version": schema.StringAttribute{
						MarkdownDescription: "version of http",
						Required:            true,
						//TODO BALANCER get from references
						Validators: []validator.String{
							stringvalidator.OneOf("h2", "http/1.1", "http/1.0"),
						},
					},
				},
			},
		},
	},
}

// SchemeModel represents the Terraform source model for the Defaults struct.
type BalancerV3DefaultsSourceModel struct {
	ClientTimeout  types.Int64 `tfsdk:"client_timeout"`
	ServerTimeout  types.Int64 `tfsdk:"server_timeout"`
	ConnectTimeout types.Int64 `tfsdk:"connect_timeout"`
}

func (m BalancerV3DefaultsSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"client_timeout":  types.Int64Type,
		"server_timeout":  types.Int64Type,
		"connect_timeout": types.Int64Type,
	}
}

var BalancerV3DefaultsScheme = map[string]schema.Attribute{
	"client_timeout": schema.Int64Attribute{
		MarkdownDescription: "Таймаут бездействия клиента в секундах",
		Optional:            true,
		Computed:            true,
		Default:             int64default.StaticInt64(30),
		Validators: []validator.Int64{
			int64validator.Between(30, 600),
		},
	},
	"server_timeout": schema.Int64Attribute{
		MarkdownDescription: "Таймаут ответа сервера в секундах",
		Computed:            true,
		Optional:            true,
		Default:             int64default.StaticInt64(30),
		Validators: []validator.Int64{
			int64validator.Between(30, 600),
		},
	},
	"connect_timeout": schema.Int64Attribute{
		MarkdownDescription: "Время ожидания клиентского соединения в секундах",
		Computed:            true,
		Optional:            true,
		Default:             int64default.StaticInt64(10),
		Validators: []validator.Int64{
			int64validator.Between(10, 600),
		},
	},
}

// SchemeModel represents the Terraform source model for the Globals struct.
type BalancerV3GlobalsSourceModel struct {
	Maxconn                           types.Int64  `tfsdk:"maxconn"`
	TuneOptions                       types.String `tfsdk:"tune_options"`
	H2WorkaroundBogusWebsocketClients types.Bool   `tfsdk:"h2_workaround_bogus_websocket_clients"`
}

func (m BalancerV3GlobalsSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"maxconn":                               types.Int64Type,
		"tune_options":                          types.StringType,
		"h2_workaround_bogus_websocket_clients": types.BoolType,
	}
}

var BalancerV3GlobalsScheme = map[string]schema.Attribute{
	"maxconn": schema.Int64Attribute{
		MarkdownDescription: "Максимальное кол-во одновременных подключений к каждому узлу балансироващика",
		Optional:            true,
		Validators: []validator.Int64{
			int64validator.AtLeast(10000),
		},
	},
	"tune_options": schema.StringAttribute{
		MarkdownDescription: "Опциональные настройки тюнинга",
		Optional:            true,
	},
	"h2_workaround_bogus_websocket_clients": schema.BoolAttribute{
		MarkdownDescription: "Отключение HTTP2 для использования websocket",
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
	},
}

type BalancerV3ConfigDataSourceModel struct {
	Backends     types.Set    `tfsdk:"backends"`
	Ports        types.Set    `tfsdk:"ports"`
	Publications types.Set    `tfsdk:"publications"`
	Globals      types.Object `tfsdk:"globals"`
	Defaults     types.Object `tfsdk:"defaults"`
}

func (m BalancerV3ConfigDataSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"backends":     types.SetType{ElemType: types.ObjectType{AttrTypes: BalancerV3BackendSourceModel{}.AttrTypes()}},
		"ports":        types.SetType{ElemType: types.ObjectType{AttrTypes: BalancerV3PortSourceModel{}.AttrTypes()}},
		"publications": types.SetType{ElemType: types.ObjectType{AttrTypes: BalancerV3PublicationSourceModel{}.AttrTypes()}},
		"globals":      types.ObjectType{AttrTypes: BalancerV3GlobalsSourceModel{}.AttrTypes()},
		"defaults":     types.ObjectType{AttrTypes: BalancerV3DefaultsSourceModel{}.AttrTypes()},
	}
}

var BalancerV3ConfigScheme = map[string]schema.Attribute{
	"backends": schema.SetNestedAttribute{
		MarkdownDescription: "Список бэкендов",
		Required:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: balancerV3BackendScheme,
		},
	},
	"ports": schema.SetNestedAttribute{
		MarkdownDescription: "Список портов",
		Required:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: BalancerV3PortScheme,
		},
	},
	"publications": schema.SetNestedAttribute{
		MarkdownDescription: "Список публикаций",
		Required:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: BalancerV3PublicationScheme,
		},
	},
	"defaults": schema.SingleNestedAttribute{
		MarkdownDescription: "Defaults",
		Required:            true,
		Attributes:          BalancerV3DefaultsScheme,
	},
	"globals": schema.SingleNestedAttribute{
		MarkdownDescription: "Globals",
		Required:            true,
		Attributes:          BalancerV3GlobalsScheme,
	},
}

func AreConfigModelsEqual(model1, model2 *BalancerV3ConfigDataSourceModel) bool {
	if model1 == nil || model2 == nil {
		return model1 == model2
	}

	sortSlice := func(slice interface{}, fieldName string) {
		value := reflect.ValueOf(slice)
		if value.Kind() != reflect.Slice {
			return
		}

		sort.SliceStable(slice, func(i, j int) bool {
			item1 := value.Index(i).FieldByName(fieldName).Interface()
			item2 := value.Index(j).FieldByName(fieldName).Interface()
			return item1.(string) < item2.(string)
		})
	}

	sortSlice(model1.Ports, "port")
	sortSlice(model2.Ports, "port")

	sortSlice(model1.Backends, "backend_name")
	sortSlice(model2.Backends, "backend_name")

	sortSlice(model1.Publications, "main_backend")
	sortSlice(model2.Publications, "main_backend")

	return reflect.DeepEqual(model1, model2)
}
