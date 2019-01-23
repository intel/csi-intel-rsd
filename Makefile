export GO111MODULE=on

build:
	@go build ./cmd/csirsd

fmt:
	@report=`gofmt -s -d -w $$(find cmd pkg -name \*.go)` ; if [ -n "$$report" ]; then echo "$$report"; exit 1; fi

vet:
	@go vet -shadow ./cmd/csirsd ./internal ./pkg/rsd 2>&1 | grep '\:' || true

lint:
	@rc=0 ; for f in $$(find -name \*.go | grep -v \.\/vendor) ; do golint -set_exit_status $$f || rc=1 ; done ; exit $$rc

all: build fmt vet lint
