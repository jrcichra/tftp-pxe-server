FROM golang:1.19.4 as builder
WORKDIR /app
COPY . .
RUN go build -v -o tftp-pxe-server cmd/server/main.go

FROM gcr.io/distroless/base-debian11:debug
WORKDIR /app
COPY --from=builder /app/tftp-pxe-server /app/tftp-pxe-server
ENTRYPOINT [ "/app/tftp-pxe-server" ]
