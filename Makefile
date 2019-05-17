export GO111MODULE=on

build:
	@go build ./cmd/csirsd

fmt:
	@report=`gofmt -s -d -w $$(find cmd pkg -name \*.go)` ; if [ -n "$$report" ]; then echo "$$report"; exit 1; fi

vet:
	@go vet ./cmd/csirsd ./internal ./pkg/rsd 2>&1 | grep '\:' || true

lint:
	@rc=0 ; for f in $$(find . -name \*.go | grep -v \.\/vendor) ; do golint -set_exit_status $$f || rc=1 ; done ; exit $$rc

test:
	@go test ./internal/ ./pkg/rsd/ -covermode=count -coverprofile=.cover.out && go tool cover -func=.cover.out

driver-image:
	@docker build -f deployments/kubernetes-1.13/driver.Dockerfile -t csi-intel-rsd-driver:devel .

all: build fmt vet lint test driver-image

.PHONY: build fmt vet lint test driver-mage all
