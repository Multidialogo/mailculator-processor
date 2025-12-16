#!/bin/sh

script_dir=$(dirname "$(realpath -s "$0")")

go mod tidy
echo '\033[1;35mRun unit tests:\033[0m'
./scripts/coverage.sh unit "$script_dir"

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
export PIPELINE_INTERVAL=1
export PIPELINE_CALLBACK_URL=http://127.0.0.1:8080
export ATTACHMENTS_BASE_PATH=testdata/attachments
export EMAIL_OUTBOX_TABLE=Outbox
export EML_STORAGE_PATH=testdata/.out/eml
export MYSQL_HOST=127.0.0.1
export MYSQL_PORT=3306
export MYSQL_USER=root
export MYSQL_PASSWORD=test
export MYSQL_DATABASE=mailculator_test
export DYNAMODB_PIPELINES_ENABLED=true
export MYSQL_PIPELINES_ENABLED=true

if ! docker compose -f "$script_dir/compose.yml" --profile test-deps up -d --build --force-recreate; then
  echo "Could not start test dependencies"
  exitCode=1
fi

if [ "${exitCode:-}" = "" ]; then
  test_packages=$(go list ./... | grep -v testutils)
  echo '\033[1;35mRun repository tests:\033[0m'
  go test $test_packages -tags=repository | grep -v '\[no test files\]'
  echo '\033[1;35mRun integration tests:\033[0m'
  go test $test_packages -tags=integration | grep -v '\[no test files\]'
fi

docker compose -f "$script_dir/compose.yml" --profile test-deps down --remove-orphans

exit $exitCode
