FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o simplelogin-mailcow-bridge main.go


FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/simplelogin-mailcow-bridge .

EXPOSE 8080

CMD ["./simplelogin-mailcow-bridge"]
