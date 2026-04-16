FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o simplelogin-mailcow-bridge main.go


FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/simplelogin-mailcow-bridge .

EXPOSE 8080

CMD ["./simplelogin-mailcow-bridge"]
