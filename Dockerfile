# Build natively and cross-compile per target: running `go build` under QEMU for
# arm64/arm-v7 made CI ~10 min. The runtime stage has no RUN, so nothing is emulated.
FROM --platform=$BUILDPLATFORM golang:1.27.3-bookworm AS builder
ARG TARGETOS TARGETARCH TARGETVARIANT
WORKDIR /app
# third_party/tftp is a `replace` target in go.mod, so it must be present for the download layer
COPY go.mod go.sum ./
COPY third_party ./third_party
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOARM=${TARGETVARIANT#v} \
    go build -v -o tftp-pxe-server cmd/server/main.go

FROM gcr.io/distroless/base-debian12:debug
WORKDIR /app
COPY --from=builder /app/tftp-pxe-server /app/tftp-pxe-server
ENTRYPOINT [ "/app/tftp-pxe-server" ]
