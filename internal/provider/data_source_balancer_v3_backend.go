package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Backend struct.
type BalancerV3BackendSourceModel struct {
	Mode               types.String `tfsdk:"mode"`
	Retries            types.Object `tfsdk:"retries"`
	Servers            types.List   `tfsdk:"servers"`
	Globalname         types.String `tfsdk:"globalname"`
	Healthcheck        types.Object `tfsdk:"healthcheck"`
	BackendName        types.String `tfsdk:"backend_name"`
	BalancingAlgorithm types.String `tfsdk:"balancing_algorithm"`
	Cookie             types.Object `tfsdk:"cookie"`
	Forwardfor         types.Object `tfsdk:"forwardfor"`
	HttpReuse          types.Object `tfsdk:"http_reuse"`
	KeepAlive          types.Object `tfsdk:"keep_alive"`
	ServersSettings    types.Object `tfsdk:"servers_settings"`
}

func (m BalancerV3BackendSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":                types.StringType,
		"retries":             types.ObjectType{AttrTypes: RetriesSourceModel{}.AttrTypes()},
		"servers":             types.ListType{ElemType: types.ObjectType{AttrTypes: ServerSourceModel{}.AttrTypes()}},
		"globalname":          types.StringType,
		"healthcheck":         types.ObjectType{AttrTypes: HealthcheckSourceModel{}.AttrTypes()},
		"backend_name":        types.StringType,
		"balancing_algorithm": types.StringType,
		"cookie":              types.ObjectType{AttrTypes: CookieSourceModel{}.AttrTypes()},
		"forwardfor":          types.ObjectType{AttrTypes: ForwardforSourceModel{}.AttrTypes()},
		"http_reuse":          types.ObjectType{AttrTypes: HttpReuseSourceModel{}.AttrTypes()},
		"keep_alive":          types.ObjectType{AttrTypes: BackendKeepAliveSourceModel{}.AttrTypes()},
		"servers_settings":    types.ObjectType{AttrTypes: ServerSettingsSourceModel{}.AttrTypes()},
	}
}

type RetriesSourceModel struct {
	Enabled    types.Bool   `tfsdk:"enabled"`
	Count      types.Int64  `tfsdk:"count"`
	Conditions types.List   `tfsdk:"conditions"`
	Redispatch types.String `tfsdk:"redispatch"`
}

func (m RetriesSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":    types.BoolType,
		"count":      types.Int64Type,
		"conditions": types.ListType{ElemType: types.StringType},
		"redispatch": types.StringType,
	}
}

type ServerSourceModel struct {
	Name      types.String `tfsdk:"name"`
	State     types.String `tfsdk:"state"`
	Address   types.String `tfsdk:"address"`
	SendProxy types.String `tfsdk:"send_proxy"`
	Maxconn   types.Int64  `tfsdk:"maxconn"`
	Weight    types.Int64  `tfsdk:"weight"`
}

func (m ServerSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":       types.StringType,
		"state":      types.StringType,
		"address":    types.StringType,
		"send_proxy": types.StringType,
		"maxconn":    types.Int64Type,
		"weight":     types.Int64Type,
	}
}

type CheckStringSourceModel struct {
	Uri       types.String `tfsdk:"uri"`
	Port      types.Int64  `tfsdk:"port"`
	Match     types.String `tfsdk:"match"`
	Value     types.String `tfsdk:"value"`
	Method    types.String `tfsdk:"method"`
	Address   types.String `tfsdk:"address"`
	UseSsl    types.String `tfsdk:"use_ssl"`
	Version   types.String `tfsdk:"version"`
	HostHdr   types.String `tfsdk:"host_hdr"`
	SendProxy types.String `tfsdk:"send_proxy"`
	Body      types.String `tfsdk:"body"`
}

func (m CheckStringSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"uri":        types.StringType,
		"port":       types.Int64Type,
		"match":      types.StringType,
		"value":      types.StringType,
		"method":     types.StringType,
		"address":    types.StringType,
		"use_ssl":    types.StringType,
		"version":    types.StringType,
		"host_hdr":   types.StringType,
		"send_proxy": types.StringType,
		"body":       types.StringType,
	}
}

type HealthcheckSourceModel struct {
	Mode         types.String `tfsdk:"mode"`
	Interval     types.Int64  `tfsdk:"interval"`
	FallCount    types.Int64  `tfsdk:"fall_count"`
	RiseCount    types.Int64  `tfsdk:"rise_count"`
	CheckStrings types.List   `tfsdk:"check_strings"`
}

func (m HealthcheckSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":          types.StringType,
		"interval":      types.Int64Type,
		"fall_count":    types.Int64Type,
		"rise_count":    types.Int64Type,
		"check_strings": types.ListType{ElemType: types.ObjectType{AttrTypes: CheckStringSourceModel{}.AttrTypes()}},
	}
}

type CookieSourceModel struct {
	Enable types.Bool `tfsdk:"enable"`
	Secure types.Bool `tfsdk:"secure"`
}

func (m CookieSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enable": types.BoolType,
		"secure": types.BoolType,
	}
}

type ForwardforSourceModel struct {
	Xff  types.Bool `tfsdk:"xff"`
	Xcip types.Bool `tfsdk:"xcip"`
	Xrip types.Bool `tfsdk:"xrip"`
}

func (m ForwardforSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"xff":  types.BoolType,
		"xcip": types.BoolType,
		"xrip": types.BoolType,
	}
}

type HttpReuseSourceModel struct {
	Mode           types.String `tfsdk:"mode"`
	PoolLowConn    types.Int64  `tfsdk:"pool_low_conn"`
	PoolMaxConn    types.Int64  `tfsdk:"pool_max_conn"`
	PoolPurgeDelay types.Int64  `tfsdk:"pool_purge_delay"`
}

func (m HttpReuseSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":             types.StringType,
		"pool_low_conn":    types.Int64Type,
		"pool_max_conn":    types.Int64Type,
		"pool_purge_delay": types.Int64Type,
	}
}

type BackendKeepAliveSourceModel struct {
	Mode  types.String `tfsdk:"mode"`
	Timer types.Int64  `tfsdk:"timer"`
}

func (m BackendKeepAliveSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"mode":  types.StringType,
		"timer": types.Int64Type,
	}
}

type ServerSettingsSourceModel struct {
	Port         types.Int64  `tfsdk:"port"`
	UseSni       types.Bool   `tfsdk:"use_sni"`
	SlowStart    types.Int64  `tfsdk:"slow_start"`
	TlsProfile   types.String `tfsdk:"tls_profile"`
	HttpSettings types.Object `tfsdk:"http_settings"`
}

func (m ServerSettingsSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"port":          types.Int64Type,
		"use_sni":       types.BoolType,
		"slow_start":    types.Int64Type,
		"tls_profile":   types.StringType,
		"http_settings": types.ObjectType{AttrTypes: HttpSettingsSourceModel{}.AttrTypes()},
	}
}

var balancerV3BackendScheme = map[string]schema.Attribute{
	"mode": schema.StringAttribute{
		MarkdownDescription: "Режим работы",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.OneOf("tcp", "http", "https"),
		},
	},
	"retries": schema.SingleNestedAttribute{
		MarkdownDescription: "Настройки повторного соединения",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Включить повторное соединение",
			},
			"count": schema.Int64Attribute{
				MarkdownDescription: "Количество попыток повторного соединения",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 20),
				},
			},
			"conditions": schema.ListAttribute{
				MarkdownDescription: "Условия повторного соединения",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 20),
					listvalidator.UniqueValues(),
				},
			},
			"redispatch": schema.StringAttribute{
				MarkdownDescription: "Повторное соединение с другим сервером",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("disabled", "every_try"),
				},
			},
		},
	},
	"servers": schema.ListNestedAttribute{
		MarkdownDescription: "Список серверов",
		Required:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Название сервера",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\.\-_]*$`),
							"must be a valid server name",
						),
						stringvalidator.LengthBetween(1, 255),
					},
				},
				"state": schema.StringAttribute{
					MarkdownDescription: "Режим работы сервера",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("active", "backup", "maintenance", "drain"),
					},
				},
				"address": schema.StringAttribute{
					MarkdownDescription: "IP адрес сервера",
					Required:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`),
							"must be a valid IP address",
						),
						stringvalidator.LengthBetween(7, 15),
					},
				},
				"send_proxy": schema.StringAttribute{
					MarkdownDescription: "Использование ProxyProtocol",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("enabled-v1", "enabled-v2", "disabled"),
					},
				},
				"maxconn": schema.Int64Attribute{
					MarkdownDescription: "Максимальное кол-во одновременных подключений к серверу",
					Required:            true,
					Validators:          []validator.Int64{},
				},
				"weight": schema.Int64Attribute{
					MarkdownDescription: "Вес сервера",
					Optional:            true,
					Computed:            true,
					Validators: []validator.Int64{
						int64validator.Between(1, 256),
					},
					Default: int64default.StaticInt64(1),
				},
			},
		},
	},
	"globalname": schema.StringAttribute{
		MarkdownDescription: "Глобальное имя публикации",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.RegexMatches(
				regexp.MustCompile(`^(([a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]\.)+[a-zA-Z]{2,63})$`),
				"must be a valid global name",
			),
			stringvalidator.LengthBetween(3, 128),
		},
	},
	"healthcheck": schema.SingleNestedAttribute{
		MarkdownDescription: "Проверки доступности",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				MarkdownDescription: "Тип проверки",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("tcp", "http"),
				},
			},
			"interval": schema.Int64Attribute{
				MarkdownDescription: "Интервал проверки в секундах",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 30),
				},
			},
			"fall_count": schema.Int64Attribute{
				MarkdownDescription: "Кол-во последовательных неудачных проверок",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
				},
			},
			"rise_count": schema.Int64Attribute{
				MarkdownDescription: "Кол-во последовательных удачных проверок",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
				},
			},
			"check_strings": schema.ListNestedAttribute{
				MarkdownDescription: "Последовательность проверок",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uri": schema.StringAttribute{
							MarkdownDescription: "URI проверкиы",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^/[a-zA-Z0-9\-\/&?=]*$`),
									"must be a valid URI",
								),
								stringvalidator.LengthBetween(1, 255),
							},
						},
						"port": schema.Int64Attribute{
							MarkdownDescription: "Порт для проверки",
							Optional:            true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65534),
							},
						},
						"match": schema.StringAttribute{
							MarkdownDescription: "Тип ожидаемого ответа от сервера",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("status", "string", "rstring", "rstatus"),
							},
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Значение ожидаемого ответа от сервера",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[^а-яА-ЯёЁ']*$`),
									"must be a valid value",
								),
								stringvalidator.LengthBetween(1, 255),
							},
						},
						"method": schema.StringAttribute{
							MarkdownDescription: "Метод проверки",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("GET", "PUT"),
							},
						},
						"address": schema.StringAttribute{
							MarkdownDescription: "Адрес проверки",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`),
									"must be a valid IP address",
								),
								stringvalidator.LengthBetween(7, 15),
							},
						},
						"use_ssl": schema.StringAttribute{
							MarkdownDescription: "Использование SSL",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("enabled", "disabled"),
							},
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Версия протокола HTTP",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^HTTP/(2|1.1|1.0)$`),
									"must be a valid HTTP version",
								),
								stringvalidator.LengthBetween(6, 8),
							},
						},
						"host_hdr": schema.StringAttribute{
							MarkdownDescription: "Заголовок хоста",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\.\-]*$`),
									"must be a valid host header",
								),
								stringvalidator.LengthBetween(1, 255),
							},
						},
						"send_proxy": schema.StringAttribute{
							MarkdownDescription: "Использовать Proxy протокол",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("enabled", "disabled"),
							},
						},
						"body": schema.StringAttribute{
							MarkdownDescription: "Тело для PUT запроса",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`[^а-яА-ЯёЁ]`),
									"must be a valid body",
								),
								stringvalidator.LengthBetween(1, 255),
							},
						},
					},
				},
			},
		},
	},
	"backend_name": schema.StringAttribute{
		MarkdownDescription: "Имя бэкенда",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.RegexMatches(
				regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\.\-_]*$`),
				"must be a valid backend name",
			),
			stringvalidator.LengthBetween(1, 255),
		},
	},
	"balancing_algorithm": schema.StringAttribute{
		MarkdownDescription: "Алгоритм распределения соединений",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.OneOf("leastconn", "roundrobin", "source"),
		},
	},
	"cookie": schema.SingleNestedAttribute{
		MarkdownDescription: "Настройки Cookie",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"enable": schema.BoolAttribute{
				MarkdownDescription: "Enable cookie persistence",
				Required:            true,
			},
			"secure": schema.BoolAttribute{
				MarkdownDescription: "Use secure cookie",
				Optional:            true,
			},
		},
	},
	"forwardfor": schema.SingleNestedAttribute{
		MarkdownDescription: "Клиентский IP в заголовках",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"xff": schema.BoolAttribute{
				MarkdownDescription: "Добавить X-Forwarded-For заголовок",
				Required:            true,
			},
			"xcip": schema.BoolAttribute{
				MarkdownDescription: "Заменить X-Client-IP заголовок",
				Optional:            true,
			},
			"xrip": schema.BoolAttribute{
				MarkdownDescription: "Заменить X-Real-IP заголовок",
				Optional:            true,
			},
		},
	},
	"http_reuse": schema.SingleNestedAttribute{
		MarkdownDescription: "Настройки повторного использования HTTP",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				MarkdownDescription: "Режим повторного использования HTTP",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("never", "safe", "aggressive", "always"),
				},
			},
			"pool_low_conn": schema.Int64Attribute{
				MarkdownDescription: "Минимальное кол-во постоянных открытых соединений",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 1000),
				},
			},
			"pool_max_conn": schema.Int64Attribute{
				MarkdownDescription: "Максимальное кол-во постоянных открытых соединений",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(-1, 10000),
				},
			},
			"pool_purge_delay": schema.Int64Attribute{
				MarkdownDescription: "Интервал очистки соединения",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 60),
				},
			},
		},
	},
	"keep_alive": schema.SingleNestedAttribute{
		MarkdownDescription: "Настройки Keep-alive",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				MarkdownDescription: "Режим работы keep-alive",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("default", "advanced", "disabled"),
				},
			},
			"timer": schema.Int64Attribute{
				MarkdownDescription: "Время жизни соединений",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
				},
			},
		},
	},
	"servers_settings": schema.SingleNestedAttribute{
		MarkdownDescription: "Настройки серверов",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"port": schema.Int64Attribute{
				MarkdownDescription: "Порт подключения к серверам",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65534),
				},
			},
			"slow_start": schema.Int64Attribute{
				MarkdownDescription: "Время медленного старта в секундах",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 600),
				},
			},
			//TODO BALANCER get from references
			"tls_profile": schema.StringAttribute{
				MarkdownDescription: "Профиль TLS",
				Optional:            true,
			},
			"http_settings": BalancerV3HttpSettingsScheme,
			"use_sni": schema.BoolAttribute{
				MarkdownDescription: "Отправка Sni",
				Optional:            true,
			},
		},
	},
}
