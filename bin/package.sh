#!/bin/bash -e

DIR=bin
BASEDIR=`dirname $0`/../..
VERSION=$1
EXTENSION="tar.gz"

if [ -d "$DIR" ] || [ -r "$DIR"/cloud-provisioner]; then
	echo "Packaging cloud-provisioner-$VERSION..."
	tar czf "$DIR"/cloud-provisioner-${VERSION}.${EXTENSION} "$DIR"/cloud-provisioner
else
	echo "Run 'make build' first"
	exit 1
fi