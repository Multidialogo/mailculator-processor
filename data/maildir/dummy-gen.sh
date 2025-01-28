#!/bin/bash

# Get the directory where the script is located
script_dir="$(dirname "$(realpath "$0")")"

# Ensure ownership of the script directory
chown -R "$(whoami):$(id -gn)" "${script_dir}"

# Define source file and target directory relative to the script's directory
source_file="${script_dir}/sample.EML.dist"
target_dir="${script_dir}/outbox"

# Cleanup directories
echo "Cleaning up directories..."
rm -rf "${script_dir}/failure"
rm -rf "${script_dir}/sent"
rm -rf "${script_dir}/outbox/users"

# Loop to copy the file 20 times with the required folder structure
for i in {1..20}
do
  # Generate random UUIDs for the folder structure
  user_uuid=$(cat /proc/sys/kernel/random/uuid)
  queue_uuid=$(cat /proc/sys/kernel/random/uuid)
  message_uuid=$(cat /proc/sys/kernel/random/uuid)

  # Create the directory structure
  message_dir="${target_dir}/users/${user_uuid}/queues/${queue_uuid}/messages"
  mkdir -p "$message_dir"

  # Define the full path of the new file
  new_file="${message_dir}/${message_uuid}.EML"

  # Copy the file to the new location
  cp "$source_file" "$new_file"

  # Optionally print the copy path
  echo "Copied to: $new_file"
done

echo "File generation complete."
