package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SchemeModel represents the Terraform source model for the Publication struct.
type BalancerV3PublicationSourceModel struct {
	Mode            types.String `tfsdk:"mode"`
	Port            types.Int64  `tfsdk:"port"`
	Globalname      types.String `tfsdk:"globalname"`
	MainBackend     types.String `tfsdk:"main_backend"`
	AliveServCount  types.Int64  `tfsdk:"alive_serv_count"`
	StandinBackends types.Set    `tfsdk:"standin_backends"`
	DefaultRouting  types.Bool   `tfsdk:"default_routing"`
	Cnames          types.List   `tfsdk:"cnames"`
	Routes          types.List   `tfsdk:"routes"`
}

func (m BalancerV3PublicationSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":             types.StringType,
		"port":             types.Int64Type,
		"globalname":       types.StringType,
		"main_backend":     types.StringType,
		"alive_serv_count": types.Int64Type,
		"standin_backends": types.SetType{ElemType: types.StringType},
		"default_routing":  types.BoolType,
		"cnames":           types.ListType{ElemType: types.StringType},
		"routes":           types.ListType{ElemType: types.ObjectType{AttrTypes: RouteModel{}.AttrTypes()}},
	}
}

type RouteModel struct {
	Uri     types.List `tfsdk:"uri"`
	Headers types.List `tfsdk:"headers"`
	Ips     types.List `tfsdk:"ips"`
}

func (m RouteModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uri":     types.ListType{ElemType: types.ObjectType{AttrTypes: EndpointModel{}.AttrTypes()}},
		"headers": types.ListType{ElemType: types.ObjectType{AttrTypes: HeaderModel{}.AttrTypes()}},
		"ips":     types.ListType{ElemType: types.StringType},
	}
}

type EndpointModel struct {
	Condition types.String `tfsdk:"condition"`
	Endpoints types.List   `tfsdk:"endpoints"`
}

func (m EndpointModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"condition": types.StringType,
		"endpoints": types.ListType{ElemType: types.StringType},
	}
}

type HeaderModel struct {
	Name      types.String `tfsdk:"name"`
	CheckType types.String `tfsdk:"check_type"`
	Value     types.String `tfsdk:"value"`
}

func (m HeaderModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":       types.StringType,
		"check_type": types.StringType,
		"value":      types.StringType,
	}
}

var BalancerV3PublicationScheme = map[string]schema.Attribute{
	"mode": schema.StringAttribute{
		MarkdownDescription: "Режим работы",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.OneOf("tcp", "http", "https"),
		},
	},
	"port": schema.Int64Attribute{
		MarkdownDescription: "Номер порта",
		Required:            true,
		Validators: []validator.Int64{
			int64validator.Between(1, 13000),
		},
	},
	"globalname": schema.StringAttribute{
		MarkdownDescription: "Имя глобальной публикации",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.RegexMatches(
				regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\.\-_]*$`),
				"must be a valid global name",
			),
			stringvalidator.LengthBetween(1, 255),
		},
	},
	"main_backend": schema.StringAttribute{
		MarkdownDescription: "Main бэкенд",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.LengthBetween(1, 255),
		},
	},
	"alive_serv_count": schema.Int64Attribute{
		MarkdownDescription: "Минимальное кол-во работоспособных серверов",
		Required:            true,
		Validators: []validator.Int64{
			int64validator.Between(1, 1),
		},
	},
	"standin_backends": schema.SetAttribute{
		MarkdownDescription: "StandIn бэкенды",
		Optional:            true,
		Computed:            true,
		ElementType:         types.StringType,
		Validators: []validator.Set{
			setvalidator.SizeAtMost(0),
		},
		PlanModifiers: []planmodifier.Set{
			DefaultEmptyStringList(),
		},
	},
	"default_routing": schema.BoolAttribute{
		MarkdownDescription: "Использование маршрута по умолчанию",
		Optional:            true,
	},
	"cnames": schema.ListAttribute{
		MarkdownDescription: "CNAMEs приложения",
		Optional:            true,
		ElementType:         types.StringType,
		Validators: []validator.List{
			listvalidator.UniqueValues(),
			listvalidator.SizeAtLeast(1),
		},
	},
	"routes": schema.ListNestedAttribute{
		MarkdownDescription: "Условия маршрутизации",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"uri": schema.ListNestedAttribute{
					MarkdownDescription: "Условия URI",
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"condition": schema.StringAttribute{
								MarkdownDescription: "Тип сопоставления",
								Optional:            true,
							},
							"endpoints": schema.ListAttribute{
								MarkdownDescription: "Endpoints",
								Required:            true,
								ElementType:         types.StringType,
								Validators: []validator.List{
									listvalidator.UniqueValues(),
									listvalidator.SizeAtLeast(1),
								},
							},
						},
					},
				},
				"headers": schema.ListNestedAttribute{
					MarkdownDescription: "Заголовки",
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								MarkdownDescription: "Названия заголовка",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^[a-z0-9-]+$`),
										"must be a valid header name",
									),
								},
							},
							"check_type": schema.StringAttribute{
								MarkdownDescription: "Тип сопоставления",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("value", "existing"),
								},
							},
							"value": schema.StringAttribute{
								MarkdownDescription: "Значение заголовка",
								Optional:            true,
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^[^\\x00-\\x1F\\x41-\\x5A\\x7F\\x80-\\xFF()<>@,;\\\"/[\\]?={} ]*$`),
										"must be a valid header value",
									),
								},
							},
						},
					},
				},
				"ips": schema.ListAttribute{
					MarkdownDescription: "IP/Network адрес",
					Optional:            true,
					ElementType:         types.StringType,
					Validators: []validator.List{
						listvalidator.UniqueValues(),
						listvalidator.SizeAtLeast(0),
					},
				},
			},
		},
	},
}
