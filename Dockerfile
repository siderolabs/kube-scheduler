FROM golang:1.21.3-alpine AS build
ADD . /src
WORKDIR /src
RUN go build .

FROM alpine:3.18.4
RUN apk add ipmitool
COPY --from=build /src/kube-scheduler /usr/local/bin/kube-scheduler
