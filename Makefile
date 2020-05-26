
SRCS != echo *go
ROOT != pwd
GO = env GOPATH=$(ROOT)/.gopath go

all: nanostatsd

nanostatsd: .gopath $(SRCS)
	$(GO) build

.gopath: webdata.go
	if [ \! -d .gopath ]; then mkdir .gopath; $(GO) get -d; fi

html:
	(cd wwwroot; echo "package main"; find . \! -type d -exec sh convert_to_go.sh "{}" \;) > ./webdata.go

clean:
	rm -f nanostatsd
	rm -rf .gopath

.PHONY: all html clean

