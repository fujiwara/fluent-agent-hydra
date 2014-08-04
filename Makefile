TARGET = cmd/fluent-agent-hydra/fluent-agent-hydra

all: test $(TARGET)

.PHONY: test
test:
	cd hydra && go test

get-deps:
	go get -d -v ./fluent/ ./hydra/

$(TARGET):
	cd cmd/fluent-agent-hydra && go build
