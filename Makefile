# SyntropyAgent-GO build script

all: agent

agent:
	go build -o sag -ldflags="-s -w" src/main.go

clean:
	go clean
	rm -f sag