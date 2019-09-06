#!/bin/sh -eu

export TEMPLATE_PATH="$(dirname $(readlink -f $0))/templates"
echo "TEMPLATE_PATH=${TEMPLATE_PATH}"
for D in $(find . -name '*_test.go' ! -path './vendor/*' | cut -f2 -d'/' | uniq); do
    echo "${D}"
    (cd ${D}; go test -v)
done
