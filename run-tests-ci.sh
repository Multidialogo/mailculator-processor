#!/bin/sh

script_dir=$(dirname "$(realpath -s "$0")")

if ! docker compose -f "$script_dir/compose.yml" --profile test-deps up -d --build --force-recreate; then
  echo "Could not start test dependencies"
  exit 1
fi

if ! docker compose -f "$script_dir/compose.yml" run --rm test; then
  echo "Test failed"
  docker compose -f "$script_dir/compose.yml" --profile test-deps down --remove-orphans
  exit 1
fi

docker compose -f "$script_dir/compose.yml" --profile test-deps down --remove-orphans
