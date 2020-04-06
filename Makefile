all: stdiotunnel

stdiotunnel: go.mod go.sum $(wildcard *.go) Dockerfile.build
	docker build --pull -t stdiotunnel_static_builder -f Dockerfile.build .
	docker run --rm stdiotunnel_static_builder | tar x
	docker rmi --no-prune stdiotunnel_static_builder

clean:
	rm -f stdiotunnel

.PHONY: all clean
