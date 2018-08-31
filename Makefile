PKG:=github.com/sapcc/omnitruck-cache
IMAGE:=sapcc/omnitruck-cache
VERSION:=0.4

build:
	go build -o bin/omnitruck-cache $(PKG)

docker:
	docker build -t $(IMAGE):$(VERSION) .

push:
	docker push $(IMAGE):$(VERSION)
