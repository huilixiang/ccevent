#output
OUTPUT=bin
BINARY=$(OUTPUT)/ccevent
VERSION=1.0.0
BUILD_TIME=`date +%Y-%m-%d`
#version info
LDFLAGS=-ldflags "-X github.com/huilixiang/ccevent/ver.Version=${VERSION} -X github.com/huilixiang/ccevent/ver.BuildTime=${BUILD_TIME}"
#source code
SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
#conf file
CONF_FILE := $(shell cp src/*.yml $(OUTPUT))
#depending pkgs
AMQP_PKGS := $(shell go get github.com/streadway/amqp)
YAML_PKGS := $(shell go get gopkg.in/yaml.v2)
LOG_PKGS := $(shell go get github.com/gogap/logrus)
.DEFAULT_GOAL: $(BINARY)
$(BINARY): $(SOURCES) $(CONF_FILE) $(AMQP_PKGS) $(YAML_PKGS) $(LOG_PKGS)
	go build ${LDFLAGS} -o ${BINARY} src/event/*.go
.PHONEY: clean
clean:
	rm -f ${BINARY}

