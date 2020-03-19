#!/bin/bash

set -x
version=0.0.1
git archive --format tar.gz --output ../v$version.tar.gz master --prefix pxesrv-$version/

/usr/bin/cp ../v$version.tar.gz /root/rpmbuild/SOURCES
rpmbuild -ba pxesrv.spec

mkdir -p ../dist
/usr/bin/cp /root/rpmbuild/RPMS/x86_64/pxesrv*.rpm /root/rpmbuild/SRPMS/pxesrv* ../dist

exit 0