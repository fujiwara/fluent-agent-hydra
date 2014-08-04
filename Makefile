TARGET = cmd/fluent-agent-hydra/fluent-agent-hydra

all: test $(TARGET)

.PHONY: test
test:
	cd hydra && go test

$(TARGET):
	cd cmd/fluent-agent-hydra && go build
