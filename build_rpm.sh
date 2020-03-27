#!/bin/bash

set -x
version=1.0.0
/usr/bin/git archive --format tar.gz --output /tmp/v$version.tar.gz master --prefix pxesrv-$version/

/usr/bin/cp /tmp/v$version.tar.gz /root/rpmbuild/SOURCES
/usr/bin/rpmbuild -ba pxesrv.spec

/usr/bin/mkdir -p /opt/dist
/usr/bin/cp /root/rpmbuild/RPMS/x86_64/pxesrv*.rpm /root/rpmbuild/SRPMS/pxesrv* /opt/dist

cd ../

tar zcf pxesrv.tgz dist/

exit 0