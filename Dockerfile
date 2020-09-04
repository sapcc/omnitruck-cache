FROM golang:1.10.2-alpine3.7

WORKDIR /go/src/github.com/sapcc/omnitruck-cache
RUN apk add --no-cache curl git make
RUN curl https://glide.sh/get | sh

ADD glide.yaml glide.lock /go/src/github.com/sapcc/omnitruck-cache/
RUN glide install -v

ADD . /go/src/github.com/sapcc/omnitruck-cache
RUN make build

FROM alpine:3.7
LABEL source_repository="https://github.com/sapcc/omnitruck-cache"
RUN apk add --no-cache curl
COPY --from=0 /go/src/github.com/sapcc/omnitruck-cache/bin/omnitruck-cache /usr/local/bin/omnitruck-cache
ENTRYPOINT ["/usr/local/bin/omnitruck-cache"]
