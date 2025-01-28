#!/bin/bash

# Get the directory where the script is located
script_dir="$(dirname "$(realpath "$0")")"

# Define source file and target directory relative to the script's directory
source_file="${script_dir}/sample_N_.EML.dist"
target_dir="${script_dir}/outbox"

# Cleanup directories
echo "Cleaning up directories..."
rm -rf "${script_dir}/failure"
rm -rf "${script_dir}/sent"
rm -rf "${script_dir}/outbox/users"

# Loop to copy the file 20 times with the required folder structure
for i in {1..5}
do
  # Generate random UUIDs for the folder structure
  user_uuid=$(cat /proc/sys/kernel/random/uuid)
  queue_uuid=$(cat /proc/sys/kernel/random/uuid)

  for i in {1..5}
  do
    message_uuid=$(cat /proc/sys/kernel/random/uuid)

    # Create the directory structure
    message_dir="${target_dir}/users/${user_uuid}/queues/${queue_uuid}/messages"
    mkdir -p "$message_dir"

   # Read the source file and replace _N_ with a random number (1, 2, or 3)
    random_num=$((RANDOM % 3 + 1))
    template_file="${source_file//_N_/${random_num}}"

    # Define the full path of the new file
    new_file="${message_dir}/${message_uuid}.EML"

    # Write the modified content to the new file
    cp "${template_file}" "${new_file}"
  done
done

echo "File generation complete."
