GIT_VER := $(shell git describe --tags)
DATE := $(shell date +%Y-%m-%dT%H:%M:%S%z)
GOPATH ?= ${HOME}/go

.PHONY: test binary all fmt clean

all: test
	go get github.com/fujiwara/fluent-agent-hydra/cmd/fluent-agent-hydra
	go get github.com/fujiwara/fluent-agent-hydra/cmd/in-forward-benchmarkd
	go get github.com/fujiwara/fluent-agent-hydra/cmd/fluent-http-tailf

install:
	cd cmd/fluent-agent-hydra && go build -ldflags "-X main.version=${GIT_VER} -X main.buildDate=${DATE}" && install fluent-agent-hydra ${GOPATH}/bin

fmt:
	go fmt ./...

clean:
	rm -f cmd/fluent-agent-hydra/fluent-agent-hydra cmd/in-forward-benchmarkd/in-forward-benchmarkd cmd/fluent-http-tailf/fluent-http-tailf pkg/*

test:
	@echo ${GOPATH}
	cd fluent && go test
	cd ltsv && go test
	cd hydra && go test

binary:
	cd cmd/fluent-agent-hydra && \
	CGO_ENABLED=0 gox -os="linux darwin windows" -arch="amd64 386" -output "../../pkg/{{.Dir}}-${GIT_VER}-{{.OS}}-{{.Arch}}" -ldflags "-X main.version=${GIT_VER} -X main.buildDate=${DATE}"
	cd pkg && find . -name "*${GIT_VER}*" -type f -exec zip {}.zip {} \;

release:
	ghr -u fujiwara -r fluent-agent-hydra -n "$(GIT_VER)" $(GIT_VER) pkg/
