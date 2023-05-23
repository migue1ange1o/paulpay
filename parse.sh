#!/bin/bash

# Check if the input file is provided
if [ -z "$1" ]; then
  echo "Usage: bash go_func_parser.sh <input_file.go>"
  exit 1
fi

input_file="$1"
output_file="functions.txt"

# Check if the input file exists
if [ ! -f "$input_file" ]; then
  echo "Error: Input file '$input_file' not found."
  exit 1
fi

# Remove the output file if it already exists
if [ -f "$output_file" ]; then
  rm "$output_file"
fi

# Parse the input file and write matching lines to the output file
while IFS= read -r line; do
  if [[ "$line" == func* ]]; then
    echo "$line" >> "$output_file"
  fi
done < "$input_file"

echo "Parsing complete. Functions written to '$output_file'."
