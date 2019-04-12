# Go builder container
FROM golang:alpine as builder
# ENV GOLANG_VERSION 1.10.6

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /go/src/github.com/seeleteam/go-seele

WORKDIR /go/src/github.com/seeleteam/go-seele

RUN make all

# Alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /go/src/github.com/seeleteam/go-seele/build /go-seele

ENV PATH /go-seele:$PATH

RUN chmod +x /go-seele/node

EXPOSE 8027 8037 8057

# start a node with your 'config.json' file, this file must be external from a volume
# For example:
#   docker run -v <your config path>:/go-seele/config:ro -it go-seele node start -c /go-seele/config/configfile
