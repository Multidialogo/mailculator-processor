#!/bin/sh

script_dir=$(dirname "$(realpath -s "$0")")

coverage() {
  temp_path="$script_dir/coverage.out"
  report_dir="$script_dir/.coverage"
  report_filename="report.html"
  report_path="$report_dir/$report_filename"

  export AWS_BASE_ENDPOINT=http://127.0.0.1:8001
  export AWS_ACCESS_KEY_ID=local
  export AWS_SECRET_ACCESS_KEY=local
  export AWS_REGION=eu-west-1
  export SMTP_HOST=127.0.0.1
  export SMTP_USER=user
  export SMTP_PASS=pass
  export SMTP_PORT=1025
  export SMTP_FROM=mailer@example.com
  export SMTP_ALLOW_INSECURE_TLS=true

  go mod tidy
  go test $(go list ./... | grep -v testutils) -coverprofile=$temp_path -p=1

  cov=$(go tool cover -func $temp_path | grep -E "^total" | grep -o -E "[0-9]*\.[0-9]*%$")
  echo "Total coverage: ${cov}"

  mkdir -p $report_dir
  go tool cover -html=$temp_path -o $report_path
  echo "Report exported at $report_path"
  rm $temp_path
}

if ! docker compose -f "$script_dir/compose.yml" --profile test-deps up -d --build --force-recreate; then
  echo "Could not start test dependencies"
  docker compose -f "$script_dir/compose.yml" --profile test-deps down --remove-orphans
  exit 1
fi

# run in subshell to avoid exporting env variables
(coverage)

docker compose -f "$script_dir/compose.yml" --profile test-deps down --remove-orphans
