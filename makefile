.PHONY: default
default:
	go build -race -ldflags="-s -w"
	./Broadcast