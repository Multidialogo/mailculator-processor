#!/bin/sh

script_dir=$(dirname "$(realpath -s "$0")")

docker compose -f "$script_dir/compose.yml" --profile test-deps up -d --build --force-recreate
trap "echo " INT
docker compose -f "$script_dir/compose.yml" run --rm test
docker compose -f "$script_dir/compose.yml" --profile test-deps down --remove-orphans -v
