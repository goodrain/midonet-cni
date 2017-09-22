midonet:
	docker run --rm -it -v `pwd`:/go/src/github.com/goodrain/midonet-cni \
	-w /go/src/github.com/goodrain/midonet-cni golang:1.8.3 go build -o _out/midonet ./cmd/midonet
host-local:
	docker run --rm -it -v `pwd`:/go/src/github.com/goodrain/midonet-cni \
	-w /go/src/github.com/goodrain/midonet-cni golang:1.8.3 go build -o _out/host-local ./cmd/host-local
portmap:
	docker run --rm -it -v `pwd`:/go/src/github.com/goodrain/midonet-cni \
	-w /go/src/github.com/goodrain/midonet-cni golang:1.8.3 go build -o _out/portmap ./cmd/portmap	
ptp:
	docker run --rm -it -v `pwd`:/go/src/github.com/goodrain/midonet-cni \
	-w /go/src/github.com/goodrain/midonet-cni golang:1.8.3 go build -o _out/ptp ./cmd/ptp
all:clean midonet host-local portmap ptp
clean:
	rm -rf _out