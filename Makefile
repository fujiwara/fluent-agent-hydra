GIT_VER := $(shell git describe --tags)
DATE := $(shell date +%Y-%m-%dT%H:%M:%S%z)

.PHONY: test get-deps binary all

all: test
	go get github.com/fujiwara/fluent-agent-hydra/cmd/fluent-agent-hydra
	go get github.com/fujiwara/fluent-agent-hydra/cmd/in-forward-benchmarkd
	go get github.com/fujiwara/fluent-agent-hydra/cmd/fluent-http-tailf

test:
	cd fluent && go test
	cd ltsv && go test
	cd hydra && go test

get-deps:
	go get -t -d -v ./fluent/ ./hydra/

binary:
	cd cmd/fluent-agent-hydra && gox -os="linux darwin windows" -arch="amd64 386" -output "../../pkg/{{.Dir}}-${GIT_VER}-{{.OS}}-{{.Arch}}" -ldflags "-X main.version ${GIT_VER} -X main.buildDate ${DATE}"
	cd pkg && find . -name "*${GIT_VER}*" -type f -exec zip {}.zip {} \;
