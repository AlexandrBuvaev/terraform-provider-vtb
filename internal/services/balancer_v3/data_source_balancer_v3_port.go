package balancerv3

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SchemeModel represents the Terraform source model for the Port struct.
type BalancerV3PortSourceModel struct {
	Mode         types.String `tfsdk:"mode"`
	Port         types.Int64  `tfsdk:"port"`
	Maxconn      types.Int64  `tfsdk:"maxconn"`
	KeepAlive    types.Object `tfsdk:"keep_alive"`
	Redirect     types.Object `tfsdk:"redirect"`
	TlsProfile   types.String `tfsdk:"tls_profile"`
	Prescription types.Object `tfsdk:"prescription"`
	HttpSettings types.Object `tfsdk:"http_settings"`
}

func (m BalancerV3PortSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":          types.StringType,
		"port":          types.Int64Type,
		"maxconn":       types.Int64Type,
		"keep_alive":    types.ObjectType{AttrTypes: PortKeepAliveModel{}.AttrTypes()},
		"redirect":      types.ObjectType{AttrTypes: RedirectModel{}.AttrTypes()},
		"tls_profile":   types.StringType,
		"prescription":  types.ObjectType{AttrTypes: PrescriptionModel{}.AttrTypes()},
		"http_settings": types.ObjectType{AttrTypes: HttpSettingsSourceModel{}.AttrTypes()},
	}
}

type PortKeepAliveModel struct {
	Tcp  types.Object `tfsdk:"tcp"`
	Http types.Object `tfsdk:"http"`
}

func (m PortKeepAliveModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"tcp":  types.ObjectType{AttrTypes: TCPKeepAliveModel{}.AttrTypes()},
		"http": types.ObjectType{AttrTypes: HTTPKeepAliveModel{}.AttrTypes()},
	}
}

type TCPKeepAliveModel struct {
	Mode  types.String `tfsdk:"mode"`
	Count types.Int64  `tfsdk:"count"`
	Timer types.Int64  `tfsdk:"timer"`
}

func (m TCPKeepAliveModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":  types.StringType,
		"count": types.Int64Type,
		"timer": types.Int64Type,
	}
}

type HTTPKeepAliveModel struct {
	Mode  types.String `tfsdk:"mode"`
	Timer types.Int64  `tfsdk:"timer"`
}

func (m HTTPKeepAliveModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":  types.StringType,
		"timer": types.Int64Type,
	}
}

type RedirectModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

func (m RedirectModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
	}
}

type PrescriptionModel struct {
	Enabled           types.Bool  `tfsdk:"enabled"`
	LiveTime          types.Int64 `tfsdk:"live_time"`
	IncludeSubdomains types.Bool  `tfsdk:"include_subdomains"`
}

func (m PrescriptionModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":            types.BoolType,
		"live_time":          types.Int64Type,
		"include_subdomains": types.BoolType,
	}
}

var BalancerV3PortScheme = map[string]schema.Attribute{
	"mode": schema.StringAttribute{
		MarkdownDescription: "Режим работы",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.OneOf("tcp", "http", "https"),
		},
	},
	"port": schema.Int64Attribute{
		MarkdownDescription: "Порт",
		Required:            true,
		Validators: []validator.Int64{
			int64validator.Between(1, 13000),
		},
	},
	"maxconn": schema.Int64Attribute{
		MarkdownDescription: "Максимальное кол-во одновременных подключений к каждому узлу балансировщика",
		Optional:            true,
		Validators:          []validator.Int64{},
	},
	"keep_alive": schema.SingleNestedAttribute{
		MarkdownDescription: "Настройки keep-alive",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"tcp": schema.SingleNestedAttribute{
				MarkdownDescription: "Настройки TCP keep-alive",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						MarkdownDescription: "Режим работы keep-alive",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("default", "advanced"),
						},
					},
					"count": schema.Int64Attribute{
						MarkdownDescription: "Кол-во keep-alive проб",
						Optional:            true,
						Validators: []validator.Int64{
							int64validator.Between(0, 100),
						},
					},
					"timer": schema.Int64Attribute{
						MarkdownDescription: "Время жизни соединений в секундах",
						Optional:            true,
						Validators: []validator.Int64{
							int64validator.Between(0, 100),
						},
					},
				},
			},
			"http": schema.SingleNestedAttribute{
				MarkdownDescription: "Настройки HTTP keep-alive",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						MarkdownDescription: "Режим работы keep-alive",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("default", "advanced"),
						},
					},
					"timer": schema.Int64Attribute{
						MarkdownDescription: "Время жизни соединений в секундах",
						Optional:            true,
						Validators: []validator.Int64{
							int64validator.Between(0, 100),
						},
					},
				},
			},
		},
	},
	"redirect": schema.SingleNestedAttribute{
		MarkdownDescription: "Настройки перенаправления HTTP на HTTPs",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Включить перенаправление с 80 на 443 порт",
				Required:            true,
			},
		},
	},
	"tls_profile": schema.StringAttribute{
		MarkdownDescription: "Профиль SSL/TLS",
		Optional:            true,
	},
	"prescription": schema.SingleNestedAttribute{
		MarkdownDescription: "Предписания клиенту подключаться по HTTPS",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Предписывать дальнейшие подключения",
				Required:            true,
			},
			"live_time": schema.Int64Attribute{
				MarkdownDescription: "Время жизни предписания в секундах",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 63072000),
				},
			},
			"include_subdomains": schema.BoolAttribute{
				MarkdownDescription: "Include subdomains",
				Optional:            true,
			},
		},
	},
	"http_settings": BalancerV3HttpSettingsScheme,
}
