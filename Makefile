build:
	docker run --rm -it -v `pwd`:/go/src/github.com/barnettzqg/midonet-cni \
	-w /go/src/github.com/barnettzqg/midonet-cni golang:1.7.3 go build -o midonet-cni