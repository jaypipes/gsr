default: test

build:
	@echo "building gsr..."
	go build

test:
	@echo "testing gsr..."
	go test -v

install:
	@echo "installing gsr"
	go install
