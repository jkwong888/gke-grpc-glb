# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

REGISTRY=gcr.io/jkwng-images/helloworld-grpc
TAG=$(shell cat version.txt)

all: proto server client

build_image: proto
	docker build -t ${REGISTRY}:${TAG} .

push_image: build_image
	docker push ${REGISTRY}:${TAG}

cert:
	openssl genrsa -out certs/ca.key 4096
	openssl req -new -x509 -key certs/ca.key -sha256 -subj "/C=US/ST=NJ/O=CA, Inc." -days 3650 -out certs/ca.cert
	openssl genrsa -out certs/service.key 4096
	openssl req -new -key certs/service.key -out certs/service.csr -config certs/certificate.conf
	openssl x509 -req -in certs/service.csr -CA certs/ca.cert -CAkey certs/ca.key -CAcreateserial -out certs/service.pem -days 3650 -sha256 -extfile certs/certificate.conf -extensions req_ext


proto: proto/helloworld
	protoc --proto_path=proto/helloworld --go_out=plugins=grpc:proto proto/helloworld/helloworld.proto 

server:
	@echo "Building server at './bin/helloworld_server' ..."
	go build -o bin/helloworld_server cmd/helloworld_server/main.go

client:
	@echo "Building client at './bin/helloworld_client' ..."
	go build -o bin/helloworld_client cmd/helloworld_client/main.go

clean:
	rm -rf ./bin

vet:
	@echo "Running vet..."
	go vet ./...

lint:
	@echo "Running golint..."
	golint ./...

setup:
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u golang.org/x/lint/golint
