FROM golang:1.17 as builder
WORKDIR /go/src/app
COPY . .
RUN make build GOOS=linux GOARCH=amd64 CGO_ENABLED=0

# @see https://console.cloud.google.com/gcr/images/distroless/GLOBAL
FROM gcr.io/distroless/static-debian11:latest-amd64
WORKDIR /opt
COPY --from=builder /go/src/app/kubernetes-controller-template ./controller
ENTRYPOINT ["/opt/controller"]
