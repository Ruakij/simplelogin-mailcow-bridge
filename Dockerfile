# Cross-compiling from the build platform, so a multi-arch build needs no
# emulated toolchain.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder

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
FROM gcr.io/distroless/static:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329

WORKDIR /app

# Matches the uid:gid of the nonroot tag; numeric so scanners and Kubernetes
# runAsNonRoot can verify it without resolving /etc/passwd.
USER 65532:65532

COPY --from=builder /app/simplelogin-mailcow-bridge .

EXPOSE 8080

ENTRYPOINT ["./simplelogin-mailcow-bridge"]
