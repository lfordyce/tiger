FROM golang:1.18-alpine as builder
WORKDIR $GOPATH/github.com/lfordyce/tiger
ADD . .
RUN apk --no-cache add git
RUN CGO_ENABLED=0 go install -a -trimpath -ldflags "-s -w -X github.com/lfordyce/tiger/pkg/consts.VersionDetails=$(date -u +"%FT%T%z")/$(git describe --always --long --dirty)"
RUN #CGO_ENABLED=0 go install -a -trimpath

FROM alpine:3.15
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 12345 -g 12345 tiger
COPY --from=builder /go/bin/tiger /usr/bin/tiger

USER 12345
WORKDIR /home/tiger
ENTRYPOINT ["tiger"]
