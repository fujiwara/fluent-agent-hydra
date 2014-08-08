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
	cd cmd/fluent-agent-hydra && gox -os="linux darwin" -arch="amd64 i386" -output "../pkg/{{.OS}}_{{.Arch}}/{{.Dir}}"
