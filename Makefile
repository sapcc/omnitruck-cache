PKG:=github.com/sapcc/omnitruck-cache
IMAGE:=sapcc/omnitruck-cache
VERSION:=0.6.2
LDFLAGS=-s -w -X main.Version=$(VERSION) -X main.GITCOMMIT=`git rev-parse --short HEAD`

build:
	go build -o bin/omnitruck-cache -ldflags="$(LDFLAGS)" $(PKG)

linux: export GOOS=linux
linux: build

docker: linux
	docker build -t keppel.eu-de-1.cloud.sap/ccloud/$(IMAGE):$(VERSION) .

push:
	docker push keppel.eu-de-1.cloud.sap/ccloud/$(IMAGE):$(VERSION)
