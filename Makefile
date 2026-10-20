LDFLAGS = -s -w

.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/simplelogin-mailcow-bridge .
