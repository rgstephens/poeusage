FROM golang:1.21-alpine AS builder

ARG VERSION=1.0.0
ARG BUILD_DATE=unknown

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/gstephens/poeusage/cmd.Version=${VERSION} -X github.com/gstephens/poeusage/cmd.BuildDate=${BUILD_DATE}" \
    -o poeusage \
    .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/poeusage /poeusage

ENTRYPOINT ["/poeusage"]
