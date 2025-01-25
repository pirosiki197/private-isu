GO_FILES := $(shell find . -name '*.go')

all: app

app: $(GO_FILES) go.mod go.sum
	go build -o app
