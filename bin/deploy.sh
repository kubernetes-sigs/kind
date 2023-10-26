#!/bin/bash -e

DIR=bin
BASEDIR=`dirname $0`/../..
VERSION=$1
EXTENSION="tar.gz"
GROUP_ID="repository.paas.cloud-provisioner"
GROUP_ID_NEXUS=${GROUP_ID//.//}
FILE="$DIR"/cloud-provisioner-${VERSION}.${EXTENSION}

if [ -d "$DIR" ] || [ -r "$FILE" ]; then
	echo "Uploading cloud-provisioner-$VERSION..."
	curl -sS -u stratio:${NEXUSPASS} --upload-file "$DIR"/cloud-provisioner-${VERSION}.${EXTENSION} http://qa.int.stratio.com/${GROUP_ID_NEXUS}/
  	echo "$GROUP_ID:cloud-provisioner:$EXTENSION" >> "$BASEDIR/deploy-recorder.lst"
  	rm -rf $BASEDIR/hack/go
else
	echo "Run 'make build' first"
	exit 1
fi

mv "$DIR"/cloud-provisioner-${VERSION}.${EXTENSION} "$DIR"/cloud-provisioner.${EXTENSION}