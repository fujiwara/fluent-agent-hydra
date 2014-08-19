GIT_VER := $(shell git describe --tags)
DATE := $(shell date +%Y-%m-%dT%H:%M:%S%z)

all: test
	go get github.com/fujiwara/fluent-agent-hydra/cmd/fluent-agent-hydra
	go get github.com/fujiwara/fluent-agent-hydra/cmd/in-forward-benchmarkd

.PHONY: test
test:
	cd hydra && go test

get-deps:
	go get -d -v ./fluent/ ./hydra/
	go get github.com/mattn/go-scan
	go get github.com/t-k/fluent-logger-golang/fluent

binary:
	cd cmd/fluent-agent-hydra && gox -os="linux darwin windows" -arch="amd64 386" -output "../../pkg/{{.OS}}_{{.Arch}}/{{.Dir}}" -ldflags "-X main.version ${GIT_VER} -X main.buildDate ${DATE}"
