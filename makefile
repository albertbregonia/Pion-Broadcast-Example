.PHONY: default
default:
	go build -race -ldflags="-w -s"
	./Broadcast