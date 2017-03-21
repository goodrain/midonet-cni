push:
	git add . && git commit -m "xxx" && git push 
build:
	docker run --rm -it -v `pwd`:/go/src/github.com/barnettzqg/midonet-cni \
	-w /go/src/github.com/barnettzqg/midonet-cni golang:1.7.3 go build -o _build/opt/cni/bin/midonet-cni
deb:build
	docker run --rm -it -v `pwd`:/midonet-cni \
	-w /midonet-cni debian:8 dpkg-deb --build _build midonet-cni-2017.03.21.deb	