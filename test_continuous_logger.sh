#!/bin/bash

echo "Starting continuous logger..."
counter=1

while true; do
    echo "[$counter] Stdout: Current time is $(date)"
    echo "[$counter] Stderr: Error message at $(date)" >&2
    counter=$((counter + 1))
    sleep 2
done