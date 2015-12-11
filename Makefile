
.PHONY: all protoc_present deps gen_all gen_pb gen_amqp build test full_coverage \
	real_line_count devserver benchmark_dev benchmark install

PROTOC := protoc -I=${GOPATH}/src:${GOPATH}/src/github.com/gogo/protobuf/protobuf/:.

RUN_PORT=5672
PERF_SCRIPT=scripts/external/perf-client/runjava.sh

all: build

protoc_present:
	which protoc

deps:
	go get github.com/boltdb/bolt
	go get github.com/gogo/protobuf/gogoproto
	go get github.com/gogo/protobuf/proto
	go get github.com/gogo/protobuf/protoc-gen-gogo
	go get github.com/rcrowley/go-metrics
	go get github.com/streadway/amqp
	go get github.com/wadey/gocovmerge
	go get golang.org/x/crypto/bcrypt

gen_all: deps gen_pb gen_amqp

gen_pb: gen_amqp protoc_present
	$(PROTOC) --gogo_out=. amqp/*.proto
	$(PROTOC) --gogo_out=. gen/*.proto

gen_amqp:
	# TODO: this requires a virtualenv to be set up with Mako. I should
	# rewrite this in go to remove the dependency.
	python scripts/amqp_gen.py
	gofmt -w amqp/*generated*.go

build: deps gen_all
	go build -o dispatchd github.com/jeffjenkins/dispatchd/server

install: deps gen_all
	go install github.com/jeffjenkins/dispatchd/server

test: deps gen_all
	go test -cover github.com/jeffjenkins/dispatchd/...

full_coverage: test
	# Output: $$GOPATH/all.cover
	python scripts/cover.py

real_line_count:
	find . | grep '.go$$' | grep -v pb.go | grep -v generated | xargs cat | wc -l

${PERF_SCRIPT}:
	mkdir -p scripts/external/
	curl -o scripts/external/perf-client.tar.gz 'https://www.rabbitmq.com/releases/rabbitmq-java-client/v3.5.6/rabbitmq-java-client-bin-3.5.6.tar.gz'
	tar -C scripts/external/ -zxf scripts/external/perf-client.tar.gz
	mv scripts/external/rabbitmq-java-client-bin-3.5.6 scripts/external/perf-client/

devserver: install
	go install github.com/jeffjenkins/dispatchd/server
	STATIC_PATH=${GOPATH}/src/github.com/jeffjenkins/dispatchd/static \
		${GOPATH}/bin/server \
		-config-file ${GOPATH}/src/github.com/jeffjenkins/dispatchd/dev/config.json

benchmark_dev: scripts/external/perf-client/runjava.sh
	RUN_PORT=1111 scripts/benchmark_helper.sh

benchmark: scripts/external/perf-client/runjava.sh
	RUN_PORT=${RUN_PORT} scripts/benchmark_helper.sh

