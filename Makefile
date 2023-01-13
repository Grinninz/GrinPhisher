TARGET=framework
PACKAGES=core database log parser

.PHONY: all
all: build

build:
	@go build -o ./bin/$(TARGET) -mod=vendor

clean:
	@go clean
	@rm -f ./bin/$(TARGET)

install:
	@mkdir -p /usr/share/framework/phishlets
	@mkdir -p /usr/share/framework/templates
	@cp ./phishlets/* /usr/share/framework/phishlets/
	@cp ./templates/* /usr/share/framework/templates/
	@cp ./bin/$(TARGET) /usr/local/bin
	
