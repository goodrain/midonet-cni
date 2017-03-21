#!/bin/bash
set -xe

program=gr-midonet-cni

base=${1:-./.release}
codename=$(lsb_release -sc)
releasedir=$base/$(lsb_release -si)/WORKDIR
rm -fr $releasedir
mkdir -p $releasedir

vers=$(git describe)

mkdir $releasedir/$program-$vers

docker run --rm -it -v `pwd`:/go/src/github.com/barnettzqg/midonet-cni \
    -w /go/src/github.com/barnettzqg/midonet-cni golang:1.7.3 go build -o $releasedir/$program-$vers/opt/cni/bin/midonet-cni midonet.go
upx --brute $releasedir/$program-$vers/opt/cni/bin/midonet-cni
wget lang.goodrain.me/public/loopback -O $releasedir/$program-$vers/opt/cni/bin/loopback
chmod +x $releasedir/$program-$vers/opt/cni/bin/loopback
cp -a debian $releasedir/$program-$vers/debian
dvers="${vers}-0"

cd $releasedir/$program-$vers
chvers=$(head -1 debian/changelog | perl -ne 's/.*\(//; s/\).*//; print')
if [ "$chvers" != "$dvers" ]; then
   DEBEMAIL="root@goodrain.com" dch -D $codename --force-distribution -b -v "$dvers" "new version"
fi

dpkg-buildpackage -us -uc
