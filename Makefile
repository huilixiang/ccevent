OUTPUT=bin
BINARY=$(OUTPUT)/couponcc
VERSION=1.0.0
#BUILD_TIME=`date '+%Y-%m-%d %H:%M:%S'`
BUILD_TIME=`date +%Y-%m-%d`
LDFLAGS=-ldflags "-X github.com/huilixiang/ccevent/ver.Version=${VERSION} -X github.com/huilixiang/ccevent/ver.BuildTime=${BUILD_TIME}"
SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
CONF_FILE := $(shell cp src/*.yml $(OUTPUT))
.DEFAULT_GOAL: $(BINARY)
$(BINARY): $(SOURCES) $(CONF_FILE)
	go build ${LDFLAGS} -o ${BINARY} src/event/*.go
.PHONEY: clean
clean:
	rm $(OUTPUT)/*
