#!/bin/bash -e
BASEDIR=`dirname $0`/..

cd $BASEDIR

CURRENT_PATH=`pwd`

export GOROOT=$CURRENT_PATH/go

export GOPATH=$CURRENT_PATH

wget -q http://tools.stratio.com/go/go1.16.linux-amd64.tar.gz

tar -xvzf go1.16.linux-amd64.tar.gz

$GOROOT/bin/go mod download github.com/go-yaml/yaml
$GOROOT/bin/go mod download github.com/BurntSushi/toml
$GOROOT/bin/go mod download github.com/alessio/shellescape
$GOROOT/bin/go mod download github.com/evanphx/json-patch/v5
$GOROOT/bin/go mod download github.com/google/safetext
$GOROOT/bin/go mod download github.com/mattn/go-isatty
$GOROOT/bin/go mod download github.com/pelletier/go-toml
$GOROOT/bin/go mod download github.com/pkg/errors
$GOROOT/bin/go mod download github.com/spf13/cobra
$GOROOT/bin/go mod download github.com/spf13/pflag
$GOROOT/bin/go mod download gopkg.in/yaml.v3
$GOROOT/bin/go mod download sigs.k8s.io/yaml

rm go1.16.linux-amd64.tar.gz

