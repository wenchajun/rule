# Copyright 2018 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

# Copyright 2018 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

FROM golang:1.17 as rule-webhook

COPY / /go/src/rule-webhook
WORKDIR /go/src/rule-webhook
ENV GOPROXY=https://goproxy.io
RUN CGO_ENABLED=0 GO111MODULE=on go build -i -ldflags '-w -s' -o rule-webhook cmd/kube-auditing-webhook/main.go

FROM kubesphere/distroless-static:nonroot

COPY --from=rule-webhook /go/src/rule-webhook/rule-webhook /usr/local/bin/

USER nonroot:nonroot

CMD ["sh"]
