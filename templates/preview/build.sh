#!/bin/sh -eu

rm -rf dist
mkdir dist

# generate version number
if [ -n "${TRAVIS_TAG:-}" ]; then
    VERSION_BASE=${TRAVIS_TAG}  
else 
    VERSION_BASE=$(git describe --abbrev=0 --tags 2> /dev/null || echo 'dblp.0.0.0')
fi
VERSION_SHORT_COMMIT=$(git rev-parse --short HEAD)
VERSION_FULL_COMMIT=$(git rev-parse HEAD)

GO_COMMON_PATH="github.com/tidepool-org/go-common"
	
echo "Build hydromail $VERSION_BASE+$VERSION_FULL_COMMIT"
go mod tidy
go build -o dist/hydromail

cp start.sh dist/

cp index.html dist/index.html
cp livecrowdin.html dist/livecrowdin.html
echo "Push email templates"
rsync -av --progress ../ dist/ --exclude '*.go' --exclude 'preview'


