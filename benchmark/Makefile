usage:
	@echo "make build && make benchmark"

build:
	cd .. && make get-deps && CGO_ENABLED=0 make install
	cp ${GOPATH}/bin/fluent-agent-hydra hydra/
	docker-compose build
	docker-compose create
	docker-compose start
	go get github.com/fujiwara/go-dummer-simple

benchmark:
	./bench.sh
