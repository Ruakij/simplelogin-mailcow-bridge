# Cross-compiling from the build platform, so a multi-arch build needs no
# emulated toolchain.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder

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
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /app/simplelogin-mailcow-bridge .

EXPOSE 8080

ENTRYPOINT ["./simplelogin-mailcow-bridge"]
