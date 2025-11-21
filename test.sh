#!/bin/sh

export TEMPLATE_PATH="$(dirname $(readlink -f $0))/templates"

echo "Running tests..."
go install github.com/jstemmer/go-junit-report@latest
go install github.com/t-yuki/gocover-cobertura@latest
echo "Tidying modules..."
go mod tidy
echo "Running tests with coverage..."
go test -v -race -coverprofile=coverage.out.tmp ./... 2>&1 > testresults.txt
echo "Generating coverage report..."
cat coverage.out.tmp | grep -v "_test" | grep -v "mocks" | grep -v "docs" > coverage.out
testPass=$?
echo "Generating test report..."
cat testresults.txt
cat testresults.txt | go-junit-report  > test-report.xml

if [ $testPass -eq 1 ]; then
  echo "Test failed"
fi
echo "Generating coverage XML..."
gocover-cobertura < coverage.out > coverage.xml
go tool cover -html='coverage.out' -o coverage.html
echo "Test completed."
exit $testPass
