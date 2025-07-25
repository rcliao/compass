#!/bin/bash

echo "Starting test logger..."
echo "This is stdout line 1"
echo "This is stderr line 1" >&2

for i in {1..5}; do
    echo "Stdout message $i at $(date)"
    echo "Stderr message $i at $(date)" >&2
    sleep 1
done

echo "Test logger finishing..."
exit 0