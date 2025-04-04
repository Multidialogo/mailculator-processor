#!/bin/sh

tags="$1"
if [ "$2" = "" ]; then
    script_dir=$(dirname "$(realpath -s "$0")")
else
    script_dir="$2"
fi

temp_path="$script_dir/coverage.out"
report_dir="$script_dir/.coverage"
report_filename="report.html"
report_path="$report_dir/$report_filename"

test_packages=$(go list ./... | grep -v testutils)
go test $test_packages -tags="$tags" -coverprofile=$temp_path

cov=$(go tool cover -func $temp_path | grep -E "^total" | grep -o -E "[0-9]*\.[0-9]*%$")

mkdir -p $report_dir
go tool cover -html=$temp_path -o $report_path
echo "Report exported at $report_path"
rm $temp_path

echo "Total coverage: $cov"