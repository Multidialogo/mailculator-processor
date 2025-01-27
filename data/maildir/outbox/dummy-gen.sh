#!/bin/bash

# Get the directory where the script is located
script_dir="$(dirname "$(realpath "$0")")"

# Define source file and target directory relative to the script's directory
source_file="$script_dir/sample.EML.dist"
target_dir="$script_dir/dummies"

# Ensure the dummies directory exists
mkdir -p "$target_dir"

# Loop to copy the file 100 times with different random directories and names
for i in {1..100}
do
  # Generate a random directory under dummies
  random_dir="$target_dir/$(cat /proc/sys/kernel/random/uuid)"

  # Create the random directory
  mkdir -p "$random_dir"

  # Generate a random name for the file
  random_name=$(cat /proc/sys/kernel/random/uuid).EML

  # Copy the file and rename it
  cp "$source_file" "$random_dir/$random_name"

  # Optionally print the copy path
  echo "Copied to: $random_dir/$random_name"
done
