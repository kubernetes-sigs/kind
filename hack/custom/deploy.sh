#!/bin/bash -e

DIR=bin
BASEDIR=`dirname $0`/../..
VERSION=`cat $BASEDIR/VERSION`
EXTENSION="tar.gz"
GROUP_ID="repository.paas.cloud-provisioner"
GROUP_ID_NEXUS=${GROUP_ID//.//}


if [ -d "$DIR" ] || [ -r "$DIR"/cloud-provisioner]; then
	echo "Uploading cloud-provisioner-$VERSION..."
	tar czf "$DIR"/cloud-provisioner-${VERSION}.${EXTENSION} "$DIR"/cloud-provisioner
	curl -sS -u stratio:${NEXUSPASS} --upload-file "$DIR"/cloud-provisioner-${VERSION}.${EXTENSION} http://qa.int.stratio.com/${GROUP_ID_NEXUS}/
  echo "$GROUP_ID:cloud-provisioner:$EXTENSION" >> "$BASEDIR/deploy-recorder.lst"
  rm -rf $BASEDIR/hack/go
else
	echo "Run 'make build' first"
	exit 1
fi