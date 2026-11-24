# Cross-compiling from the build platform, so a multi-arch build needs no
# emulated toolchain.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w" -o simplelogin-mailcow-bridge main.go


# The binary is static; the image only needs CA certificates for the HTTPS and
# IMAPS connections to Mailcow, which distroless/static ships.
FROM gcr.io/distroless/static:nonroot@sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5

WORKDIR /app

# Matches the uid:gid of the nonroot tag; numeric so scanners and Kubernetes
# runAsNonRoot can verify it without resolving /etc/passwd.
USER 65532:65532

COPY --from=builder /app/simplelogin-mailcow-bridge .

EXPOSE 8080

ENTRYPOINT ["./simplelogin-mailcow-bridge"]
