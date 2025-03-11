#!/bin/sh

go run gotest.tools/gotestsum@latest ./... -coverpkg=./... -coverprofile=coverage.out

cov=$(go tool cover -func coverage.out | grep -E "^total" | grep -o -E "[0-9]*\.[0-9]*%$")
echo "Total coverage: ${cov}"

mkdir -p ./.coverage

go tool cover -html=coverage.out -o ./.coverage/report.html
echo "Report exported at .coverage/report.html"

rm ./coverage.out
