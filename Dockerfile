FROM alpine:3.20 AS base
RUN apk --no-cache add ca-certificates
RUN mkdir -p /tmp/cache_tmp

FROM scratch
LABEL source_repository="https://github.com/sapcc/omnitruck-cache"

COPY --from=base  /tmp/cache_tmp /tmp
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY bin/omnitruck-cache /omnitruck-cache

ENTRYPOINT [ "/omnitruck-cache" ]