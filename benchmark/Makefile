usage:
	@echo "make build && make benchmark"

build:
	cd .. && make get-deps && make install
	cp ${GOPATH}/bin/fluent-agent-hydra hydra/
	docker-compose build
	docker-compose create
	docker-compose start
	go get github.com/fujiwara/go-dummer-simple

benchmark:
	./bench.sh /data
