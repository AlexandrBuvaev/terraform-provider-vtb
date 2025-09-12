
TEST?=$$(go list ./... | grep -v 'vendor')
HOSTNAME=vtb
NAMESPACE=vtb-cloud
NAME=vtb
BINARY=terraform-provider-${NAME}
VERSION=2.16.0
OS_ARCH=linux_amd64

default: build

build:
	CGO_ENABLED=0 go build -o ${BINARY}

release: release_linux release_windows release_macos_m1

release_linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_linux_amd64
	zip -j ./bin/${BINARY}_${VERSION}_linux_amd64.zip ./bin/${BINARY}_${VERSION}_linux_amd64

release_windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_windows_amd64
	zip -j ./bin/${BINARY}_${VERSION}_windows_amd64.zip ./bin/${BINARY}_${VERSION}_windows_amd64

release_macos_m1:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./bin/macos/${BINARY}_${VERSION}_darwin_arm64
	zip -j ./bin/macos/${BINARY}_${VERSION}_darwin_arm64.zip ./bin/macos/${BINARY}_${VERSION}_darwin_arm64

release_macos_intel:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/macos/${BINARY}_${VERSION}_darwin_amd64
	zip -j ./bin/macos/${BINARY}_${VERSION}_darwin_amd64.zip ./bin/macos/${BINARY}_${VERSION}_darwin_amd64

install_local: build
	rm -f ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}/${BINARY}
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}/
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	rm -f .terraform.lock.hcl

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

generate_doc:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

testacc_references:
	TF_ACC=1 go test -run=TestAccReference $(TEST) -v $(TESTARGS) -timeout 60m -v

testacc_compute:
	TF_ACC=1 go test -run=TestAccCompute $(TEST) -v $(TESTARGS) -timeout 60m -v

testacc_kafka:
	TF_ACC=1 go test -run=TestAccKafka $(TEST) -v $(TESTARGS) -timeout 60m -v
