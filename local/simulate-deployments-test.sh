#!/bin/sh

script_dir=$(dirname $(realpath -s $0))

root_dir="$script_dir/.."

sh "$root_dir/deployments/start-test-dependencies.sh"

sh "$root_dir/deployments/test.sh"

sh "$root_dir/deployments/stop-test-dependencies.sh"

chown 1000:1000 -R "$root_dir"
