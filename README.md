# Terraform-provider VTB Cloud

Terraform — программное обеспечение с открытым исходным кодом, используемое для управления внешними ресурсами (например, в рамках модели инфраструктура как код). Создано и поддерживается компанией HashiCorp. Пользователи определяют инфраструктуру с помощью декларативного языка конфигурации, известного как HashiCorp Configuration Language (HCL).

Terraform провайдер позволяет управлять множеством ресурсов в VTB Cloud. Пользователи могут взаимодействовать с VTB Cloud, объявляя ресурсы(resources) или вызывая источники данных(data sources).

## Requirements

- [Terraform Core](https://developer.hashicorp.com/terraform/downloads) >= 1.6
- [Go](https://go.dev/doc/install) >= 1.20

- [Документация](#документация)
- [Начало работы](#начало-работы)
    - [Установка](#установка)
    - [Получение API-ключа](#получение-api-ключа)
    - [Пример использования](#пример-использования)
    - [Удаление созданных ресурсов](#удаление-созданных-ресурсов)
- [Разработка](#разработка)
    - [Рабочее пространство (go workspace)](#рабочее-пространство-go-workspace)
    - [Генерация документации](#генерация-документации)
    - [Рекомендации по настройке среды разработки провайдера](#рекомендации-по-настройке-среды-разработки-провайдера)
    - [Тестирование и отладка](#тестирование-и-отладка)
    - [TODO](#todo)

## Документация

- Ресурсы:
    - [access_group_instance](docs/resources/access_group_instance.md) - Группы доступа
    - [agent_orchestration_instance](docs/resources/agent_orchestration_instance.md) - Агент Оркестрации
    - [airflow_cluster](docs/resources/airflow_cluster.md) - Airflow Cluster
    - [airflow_standalone](docs/resources/airflow_standalone.md) - Airflow Standalone
    - [artemis_cluster](docs/resources/artemis_cluster.md) - Кластер VTB Artemis
    - [artemis_address_policy](docs/resources/artemis_address_policy.md) - Адрессная политика VTB Artemis
    - [artemis_tuz](docs/resources/artemis_tuz.md) - ТУЗ VTB Artemis
    - [balancer_v3_cluster](docs/resources/balancer_v3_cluster.md) - Load Balancer v3
    - [clickhouse_cluster](docs/resources/clickhouse_cluster.md) - ClickHouse Cluster
    - [clickhouse_instance](docs/resources/clickhouse_instance.md) - ClickHouse
    - [compute_instance](docs/resources/compute_instance.md) - Базовые вычисления: Astra Linux
    - [etcd_instance](docs/resources/etcd_instance.md) - Кластер ETCD
    - [grafana_instance](docs/resources/grafana_instance.md) - Grafana
    - [k8sproject_instance](docs/resources/k8sproject_instance.md) - K8s project
    - [ktaas_instance](docs/resources/ktaas_instance.md) - Kafka Topic как сервис
    - [sync_xpert_cluster](docs/resources/sync_xpert_cluster.md) - Кластер Sync Xpert
    - [sync_xpert_connector](docs/resources/sync_xpert_connector.md) - Коннекторы Sync Xpert
    - [kafka_instance](docs/resources/kafka_instance.md) - Apache Kafka Cluster Astra
    - [nginx_instance](docs/resources/nginx_instance.md) - Nginx Astra
    - [open_messaging_instance](docs/resources/open_messaging_instance.md) - OpenMessaging Astra
    - [postgresql_instance](docs/resources/postgresql_instance.md) - PostgreSQL (Astra Linux), PostgreSQL Cluster Astra Linux
    - [rabbitmq_cluster](docs/resources/rabbitmq_cluster.md) - Кластер RabbitMQ
    - [rabbitmq_vhosts](docs/resources/rabbitmq_vhosts.md) - Виртуальные хосты RabbitMQ
    - [rabbitmq_user](docs/resources/rabbitmq_user.md) - Пользователи виртуальных хостов RabbitMQ
    - [redis_instance](docs/resources/redis_instance.md) - Redis Astra
    - [redis_sentinel_instance](docs/resources/redis_sentinel_instance.md) - Redis Sentinel (Redis с репилкацией)
    - [rqaas_instance](docs/resources/rqaas_instance.md) - RabbitMQ Очередь как сервис
    - [tarantool_cluster](docs/resources/tarantool_cluster.md) - Tarantool Data Grid v2/Tarantool Enterprise v2
    - [wildfly_instance](docs/resources/wildfly_instance.md) - WildFly Astra
    
- Источники данных:
    - [user_data](docs/data-sources/user_data.md) - Пользователи
    - [core_data](docs/data-sources/core_data.md) - Конфгурация базовых параметров заказа
    - [flavor_data](docs/data-sources/flavor_data.md) - Конфигурация флейвора
    - [cluster_layout](docs/data-sources/cluster_layout.md) - Конфигурация кластерного продукта
    - [artemis_image_data](docs/data-sources/artemis_image_data.md) - Образ для VTB Artemis
    - [agent_orchestration_image_data](docs/data-sources/agent_orchestration_image_data.md) - Образ для Агента Оркестрации
    - [airflow_image_data](docs/data-sources/airflow_image_data.md) - Образ для Airflow Cluster/Airflow Standalone
    - [balancer_v3_image_data](docs/data-sources/balancer_v3_image_data.md) - Образ для Load Balancer v3
    - [clickhouse_cluster_image_data](docs/data-sources/clickhouse_cluster_image_data.md) - Образ для ClickHouse Cluster
    - [clickhouse_image_data](docs/data-sources/clickhouse_image_data.md) - Образ для ClickHouse
    - [compute_image_data](docs/data-sources/compute_image_data.md) - Образ для базовых вычислений
    - [etcd_image_data](docs/data-sources/etcd_image_data.md) - Образ для ETCD
    - [grafana_image_data](docs/data-sources/grafana_image_data.md) - Образ для Grafana
    - [jenkins_agent_susbsystem_data](docs/data-sources/jenkins_agent_subsystem_data.md) - Схема данных для подсистемы агента Jenkins
    - [kafka_image_data](docs/data-sources/kafka_image_data.md) - Образ для Apache Kafka Cluster
    - [debezium_image_data](docs/data-sources/debezium_image_data.md) - Образ для кластера VTB Debezium
    - [nginx_image_data](docs/data-sources/nginx_image_data.md) - Образ для Nginx
    - [open_messaging_image_data](docs/data-sources/open_messaging_image_data.md) - Образ для OpenMessaging
    - [postgresql_image_data](docs/data-sources/postgresql_image_data.md) - Образ для PostgreSQL
    - [rabbitmq_image_data](docs/data-sources/rabbitmq_image_data.md) - Образ для RabbitMQ
    - [redis_image_data](docs/data-sources/redis_image_data.md) - Образ для Redis
    - [redis_sentinel_image_data](docs/data-sources/redis_sentinel_image_data.md) - Образ для Redis Sentinel
    - [rqaas_cluster_data](docs/data-sources/rqaas_cluster_data.md) - Основные параметры кластера для RQaaS
    - [tdg_image_data](docs/data-sources/tdg_image_data.md) - Образ для Tarantool Data Grid v2
    - [te_image_data](docs/data-sources/te_image_data.md) - Образ для Tarantool Enterprise v2
    - [wildfly_image_data](docs/data-sources/wildfly_image_data.md) - Образ для Wildfly

## Начало работы

### Установка

1. Проверьте установленный Terraform. Подразумевается, что у вас уже установлен `Terraform`, проверить это можно с помощью:
Если `Terraform` у вас ещё не установлен, то вы можете выполнить инсталяцию по [инструкции с официального сайта](https://developer.hashicorp.com/terraform/install).
```shell
λ terraform -version
Terraform v1.7.0-dev
on linux_amd64
```
2. Создайте директорию
    - Для Windows: `%appdata%\terraform.d\plugins\vtb\vtb-cloud\vtb\${VERSION}\windows_amd64`
    - Для Linux и Darwin: `~/.terraform.d/plugins/vtb/vtb-cloud/vtb/${VERSION}/{linux|darwin}_amd64`
3. Переместите в данную директорию [исполняемый файл](#сборка) `terraform-provider-vtb_${VERSION}_${OS_ARCH}`:
```shell
mv terraform-provider-vtb_${VERSION}_${OS_ARCH} ~/.terraform.d/plugins/vtb/vtb-cloud/vtb/${VERSION}/${OS_ARCH}
```
4. В любом удобном месте создайте директорию, в которой будет находиться описание вашей инфраструктуры
5. Создайте в директории файл `main.tf` с минимальным содержанием: 
```hcl
terraform {
  required_providers {
    vtb = {
      source  = "vtb/vtb-cloud/vtb"
    }
  }
  required_version = ">= 0.13"
}
```
5. Выполните `terraform init`; в случае успешной инициализации провайдера вас ожидает:
```
Terraform has been successfully initialized!
```

### Получение API-ключа

Для дальнейшей работы с порталом VTB Cloud необходимо создать сервисный аккаунт в рамках вашего проекта
и сгенерировать новый API-ключ. Реквизиты созданного API-ключа используются для первоначальной авторизации провайдера:

```hcl
provider "vtb" {
  client_id     = "Идентификатор ключа"
  client_secret = "Сам ключ"
  project_name  = "Идентификатор вашего проекта"
}
```


### Пример использования

```hcl
data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  domain      = "corp.dev.vtb"
  net_segment = "dev-srv-app"
  zone        = "msk-north"
}

data "vtb_compute_image_data" "name" {
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_flavor_data" "c2m4" {
  cores  = 2
  memory = 4
}

resource "vtb_compute_instance" "name" {
  lifetime = 2
  label    = "TerraformComputeAstra1"
  core     = data.vtb_core_data.dev
  flavor   = data.vtb_flavor_data.c2m4
  image    = data.vtb_compute_image_data.name
  extra_mounts = {
    "/app" = {
      size        = 10
    }
  }
  access = {
    "superuser" = [
      "example-group-name",
    ],
  }
}

```

Для проверки корректности составленной конфигурации выполнить команду:
```shell
terraform validate
```

Для планирование ожидаемой инфраструктуры выполнить команду:
```shell
terraform plan
```

Для применения составленной конфигурацию выполнить команду:
```shell
terraform apply
```

Прежде чем Terraform начнет применять конфигурацию, он сначала представит план предполагаемых изменений и запросит ваше ручное подтверждение: 
```shell
Do you want to perform these actions?
  Terraform will perform the actions described above.
  Only 'yes' will be accepted to approve.

  Enter a value:
```

Пишем `yes` и нажимаем Enter.

После выполнения `apply` Terraform выведет результат применения конфигурации, который можно проверить в проекте вашего аккаунта на портале VTB.Cloud.

### Удаление созданных ресурсов 

Чтобы удалить все ресурсы, созданные через Terraform выполните команду;
```shell
terraform destroy -target RESOURCE_TYPE.NAME
```

Во время выполнения будет выведен план удаления существующего ресурса и Terraform запросит ручное подтверждение перед началом своей работы:
```shell
Do you really want to destroy all resources?
  Terraform will destroy all your managed infrastructure, as shown above.
  There is no undo. Only 'yes' will be accepted to confirm.

  Enter a value:
```
Введите слово `yes` и нажмите *Enter*.



## Разработка


### Blue стед
Для переключения API на стенд Blue выполните:
```shell
export PORTAL_STAND=blue
```
При установнки данного значения все запросы будут адресоваться к стенду Blue


### Сборка

```shell
make install_local && terraform init
```

### Создание документации
```shell
make generate_doc     
```

### Рекомендации

Добавьте bitbucket.region.vtb.ru в переменную окружения GOPRIVATE, чтобы go get не пытался ходить в приватные репозитории через публичный GOPROXY. Средствами IDE (рекомендуется VSCode или VSCodium), либо через CLI:

```shell
go env -w GORIVATE=bitbucket.region.vtb.ru
```

Чтобы комманда `go get` могла ходить в наши приватные репозитории добавьте замену URL в глобальный конфиг GIT

```shell
git config --global url."https://$(username):$(user_password)@bitbucket.region.vtb.ru".insteadOf."https://bitbucket.region.vtb.ru"
```

Создайте каталог для нового Go workspace и cклонируйте данный репозиторий [terraform-provider-vtb](https://bitbucket.region.vtb.ru/scm/puos/terraform-provider-vtb.git) и [go-cloud-api](https://bitbucket.region.vtb.ru/scm/puos/go-cloud-api.git) в один каталог, в котором не должно быть других проектов, особенно модулей Go.
После клонирования обоих репозиториев в общем каталоге создайте Go workspace и добавьте в него оба репозитория.

```shell
mkdir vtb_terraform
cd vtb_terraform
git clone https://bitbucket.region.vtb.ru/scm/puos/terraform-provider-vtb.git
git clone https://bitbucket.region.vtb.ru/scm/puos/go-cloud-api.git
go work init
go work use ./terraform-provider-vtb
go work use ./go-cloud-api
```

### Тестирование и отладка

Скачайте архив с исполняемым файлом terraform для [Windows](https://storage.cloud.vtb.ru/product/terraform_1.2.4_windows_amd64.zip) или [Linux](https://storage.cloud.vtb.ru/product/terraform_1.2.4_windows_amd64.zip) и распакуйте его в каталог созаднного workspace, либо в любой подходящий каталог доступный через `${PATH}`. Скопируйте пример файла настроек terraform cli в домашний каталог пользователя.

```shell
cp ./terraform-provider-vtb/examples/terraform.rc ${HOME}/ # linux
cp .\terraform-provider-vtb\examples\terraform.rc %appdata%\ # Windows
```

После внесения изменений в модули и сборки терраформ провайдера, для упрощения и ускорения тестирования до публикации изменений можно раскоментировать в ранее скопированном файле настроек (terraform.rc) директиву `devevelop_overrides` и указать полный путь до исполяемого файла скомпилированного провайдера (без расширения имени ".exe" даже на windows). Это позволит запускать терраформ с разрабатываемым провайдером без необходимости его переустановки/обновления и пропустить процедуры верификации провайдера, что позволяет значительно ускорить тестирование и отладку провайдера.

### TODO
- Написание Acceptance тестов на ресуры
- Реализовать запуск тестов в пайплайне
- Реализовать сборку и сохранение бинарников в артефакты после мержа в мейн
- Реализовать авто-генерацию документацию с помощью пре-коммит хука