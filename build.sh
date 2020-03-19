#!/bin/bash

set -x
version=0.0.1
/usr/bin/git archive --format tar.gz --output /tmp/v$version.tar.gz master --prefix pxesrv-$version/

/usr/bin/cp /tmp/v$version.tar.gz /root/rpmbuild/SOURCES
/usr/bin/rpmbuild -ba pxesrv.spec

/usr/binmkdir -p /tmp/dist
/usr/bin/cp /root/rpmbuild/RPMS/x86_64/pxesrv*.rpm /root/rpmbuild/SRPMS/pxesrv* /tmp/dist

exit 0