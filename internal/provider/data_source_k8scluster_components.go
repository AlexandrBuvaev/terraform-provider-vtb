package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type K8sClusterComponentsModel struct {
	Istio           *K8sClusterIstioModel `tfsdk:"istio"`
	Astrom          *K8sClusterFlagModel  `tfsdk:"astrom"`
	GpuOperator     *K8sClusterFlagModel  `tfsdk:"gpu_operator"`
	TsamOperator    *K8sClusterFlagModel  `tfsdk:"tsam_operator"`
	TridentOperator *K8sClusterFlagModel  `tfsdk:"trident_operator"`
	ChaosMesh       *K8sClusterFlagModel  `tfsdk:"chaos_mesh"`
	TsdsOperator    *K8sClusterFlagModel  `tfsdk:"tsds_operator"`
	TslgOperator    *K8sClusterFlagModel  `tfsdk:"tslg_operator"`
	Tyk             *K8sClusterFlagModel  `tfsdk:"tyk"`
}

type K8sClusterIstioModel struct {
	Options       types.Object                       `tfsdk:"options"`
	Installed     types.Bool                         `tfsdk:"installed"`
	ControlPlanes []K8sClusterIstioControlPlaneModel `tfsdk:"control_planes"`
	OptionsFlavor types.String                       `tfsdk:"options_flavor"`
}

type K8sClusterIstioOptionsModel struct {
	ExtauthOperator     types.Bool `tfsdk:"extauth_operator"`
	GatewayOperator     types.Bool `tfsdk:"gateway_operator"`
	AccesslogsOperator  types.Bool `tfsdk:"accesslogs_operator"`
	RatelimiterOperator types.Bool `tfsdk:"ratelimiter_operator"`
	SmOperator          types.Bool `tfsdk:"sm_operator"`
	SpanOperator        types.Bool `tfsdk:"span_operator"`
}

type K8sClusterIstioControlPlaneModel struct {
	Name    types.String                  `tfsdk:"name"`
	Flavor  types.String                  `tfsdk:"flavor"`
	Options K8sClusterIstioCPOptionsModel `tfsdk:"options"`
}

type K8sClusterIstioCPOptionsModel struct {
	Eventrouter types.Bool `tfsdk:"eventrouter"`
}

type K8sClusterFlagModel struct {
	Installed types.Bool `tfsdk:"installed"`
}

var K8sClusterComponentsSchema = map[string]schema.Attribute{
	"istio": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компонент Istio",
		Attributes:          K8sClusterIstioSchema,
	},
	"astrom": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компонент Ключ-Астром",
		Attributes: map[string]schema.Attribute{
			"installed": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Флаг подключения Ключ-Астром",
			},
		},
	},
	"gpu_operator": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компонент GPU оператор",
		Attributes: map[string]schema.Attribute{
			"installed": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Флаг подключения GPU оператора",
			},
		},
	},
	"trident_operator": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компонент Trident оператор",
		Attributes: map[string]schema.Attribute{
			"installed": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Флаг подключения Trident оператора",
			},
		},
	},
	"tsam_operator": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компонент TSAM",
		Attributes: map[string]schema.Attribute{
			"installed": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Флаг подключения TSAM",
			},
		},
	},
	"chaos_mesh": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компонент Chaos Mesh",
		Attributes: map[string]schema.Attribute{
			"installed": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Флаг подключения Chaos Mesh",
			},
		},
	},
	"tsds_operator": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компонент TSDS",
		Attributes: map[string]schema.Attribute{
			"installed": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Флаг подключения TSDS",
			},
		},
	},
	"tslg_operator": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компонент TSLG",
		Attributes: map[string]schema.Attribute{
			"installed": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Флаг подключения TSLG",
			},
		},
	},
	"tyk": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компонент TYK",
		Attributes: map[string]schema.Attribute{
			"installed": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Флаг подключения TYK",
			},
		},
	},
}

var K8sClusterIstioSchema = map[string]schema.Attribute{
	"installed": schema.BoolAttribute{
		Required:            true,
		MarkdownDescription: "Флаг подключения Istio",
	},
	"options_flavor": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Размер операторов Istio",
	},
	"options": schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "Операторы Istio",
		Attributes:          K8sClusterIstioOptionSchema,
	},
	"control_planes": schema.ListNestedAttribute{
		MarkdownDescription: "Список контрольных панелей Istio",
		Required:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: K8sClusterIstioControlPlanesSchema,
		},
	},
}

var K8sClusterIstioOptionSchema = map[string]schema.Attribute{
	"extauth_operator": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг подключения ExtauthOperator",
	},
	"gateway_operator": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг подключения GatewayOperator",
	},
	"accesslogs_operator": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг подключения AccesslogsOperator",
	},
	"ratelimiter_operator": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг подключения RatelimiterOperator",
	},
	"sm_operator": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг подключения SM Operator",
	},
	"span_operator": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг подключения SPAN Operator",
	},
}

var K8sClusterIstioControlPlanesSchema = map[string]schema.Attribute{
	"name": schema.StringAttribute{
		MarkdownDescription: "Название. Становится известно после создания заказа",
		Computed:            true,
		Optional:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"flavor": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Размер контрольной панели Istio",
	},
	"options": schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "Параметры Istio",
		Attributes: map[string]schema.Attribute{
			"eventrouter": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Флаг подключения Eventrouter",
			},
		},
	},
}
