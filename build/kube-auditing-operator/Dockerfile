FROM golang:1.17 as rule-operator

COPY / /go/src/rule
WORKDIR /go/src/rule
ENV GOPROXY=https://goproxy.io
RUN CGO_ENABLED=0 GO111MODULE=on go build -i -ldflags '-w -s' -o rule-operator cmd/rule/main.go

FROM kubesphere/distroless-static:nonroot

COPY --from=rule-operator /go/src/rule/rule-operator /usr/local/bin/

USER nonroot:nonroot

CMD ["sh"]
