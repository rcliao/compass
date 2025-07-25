#!/bin/bash

echo "DEBUG: Starting debug logger"
echo "DEBUG: Current time is $(date)"
echo "ERROR: This goes to stderr" >&2

for i in {1..3}; do
    echo "DEBUG: Message $i"
    echo "ERROR: Error $i" >&2
    sleep 1
done

echo "DEBUG: Finishing debug logger"