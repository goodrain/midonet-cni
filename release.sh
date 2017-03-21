#!/bin/bash
set -xe

program=gr-midonet-cni

base=${1:-./.release}
codename=$(lsb_release -sc)
releasedir=$base/$(lsb_release -si)/WORKDIR
rm -fr $releasedir
mkdir -p $releasedir

vers=0.1.0

mkdir $releasedir/$program-$vers
cp -a debian $releasedir/$program-$vers/debian
cp -a opt $releasedir/$program-$vers/opt
dvers="${vers}-0"

cd $releasedir/$program-$vers
chvers=$(head -1 debian/changelog | perl -ne 's/.*\(//; s/\).*//; print')
if [ "$chvers" != "$dvers" ]; then
   DEBEMAIL="root@goodrain.com" dch -D $codename --force-distribution -b -v "$dvers" "new version"
fi

dpkg-buildpackage -us -uc
