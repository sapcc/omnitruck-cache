PKG:=github.com/sapcc/omnitruck-cache
IMAGE:=sapcc/omnitruck-cache
VERSION:=0.1

build:
	go build -o bin/omnitruck-cache $(PKG)

docker:
	docker build -t $(IMAGE):$(VERSION) .