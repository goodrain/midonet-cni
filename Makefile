midonet:
	docker run --rm -it -v `pwd`:/go/src/github.com/goodrain/midonet-cni \
	-w /go/src/github.com/goodrain/midonet-cni golang:1.8.3 go build -o _out/midonet ./cmd/midonet
all:clean midonet
clean:
	rm -rf _out