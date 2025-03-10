#!/bin/sh

script_dir=$(dirname "$(realpath -s "$0")")

docker compose -f "$script_dir/docker/compose.yml" --profile develop down --remove-orphans -v
