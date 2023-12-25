# Copyright 2023 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

# Copyright 2023 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by a Apache license
# that can be found in the LICENSE file.

FROM golang:1.18 as whizard-telemetry-ruler-builder

COPY / /go/src/whizard-telemetry-ruler
WORKDIR /go/src/whizard-telemetry-ruler
ENV GOPROXY=https://goproxy.io
RUN CGO_ENABLED=0 GO111MODULE=on go build -i -ldflags '-w -s' -o whizard-telemetry-ruler cmd/whizard-telemetry-ruler/main.go

FROM kubesphere/distroless-static:nonroot

COPY --from=whizard-telemetry-ruler-builder /go/src/whizard-telemetry-ruler/whizard-telemetry-ruler /usr/local/bin/

USER nonroot:nonroot

CMD ["sh"]
