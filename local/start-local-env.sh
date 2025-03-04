#!/bin/sh

script_dir=$(dirname $(realpath -s $0))

docker compose -f "$script_dir/docker/compose.yml" --profile develop up -d --build --force-recreate

docker compose -f "$script_dir/docker/compose.yml" run --rm init-db > /dev/null
