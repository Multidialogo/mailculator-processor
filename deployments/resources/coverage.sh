#!/bin/sh

temp_filename='coverage.out'
report_dir='.coverage'
report_filename='report.html'
report_path="$report_dir/$report_filename"

go mod tidy

go run gotest.tools/gotestsum@latest ./... -coverpkg=./... -coverprofile=$temp_filename

cov=$(go tool cover -func $temp_filename | grep -E "^total" | grep -o -E "[0-9]*\.[0-9]*%$")
echo "Total coverage: ${cov}"

mkdir -p "./$report_dir"

go tool cover -html=$temp_filename -o "./$report_path"
echo "Report exported at $report_path"

rm "./$temp_filename"
