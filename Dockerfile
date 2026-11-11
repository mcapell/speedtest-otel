FROM --platform=$BUILDPLATFORM golang:1.27@sha256:01aa404fa56b604b6f48e90c1d6e2c79fe5cd78f9592f78d80303fbe171cdcc0 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o speedtest .

FROM scratch
WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/speedtest .
ENTRYPOINT ["./speedtest"]
