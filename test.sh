#!/bin/sh

export TEMPLATE_PATH="$(dirname $(readlink -f $0))/templates"

go install github.com/jstemmer/go-junit-report@latest
go install github.com/t-yuki/gocover-cobertura@latest
go mod tidy
go test -v -race -coverprofile=coverage.out.tmp ./... 2>&1 > testresults.txt
cat coverage.out.tmp | grep -v "_test" | grep -v "mocks" | grep -v "docs" > coverage.out
testPass=$?
cat test-report.txt
cat test-report.txt | go-junit-report  > test-report.xml

if [ $testPass -eq 1 ]; then
  echo "Test failed"
fi

gocover-cobertura < coverage.out > coverage.xml
go tool cover -html='coverage.out' -o coverage.html

exit $testPass
