#!/bin/bash

PROCESS_ID="edf2d905"

echo "Starting process..."
echo "compass.process.start {\"id\": \"$PROCESS_ID\"}" | ./bin/compass --cli

sleep 3

echo -e "\n\nChecking logs after 3 seconds..."
echo "compass.process.logs {\"id\": \"$PROCESS_ID\", \"limit\": 10}" | ./bin/compass --cli | grep -v "compass>"

sleep 3

echo -e "\n\nChecking logs after 6 seconds..."
echo "compass.process.logs {\"id\": \"$PROCESS_ID\", \"limit\": 15}" | ./bin/compass --cli | grep -v "compass>"

echo -e "\n\nStopping process..."
echo "compass.process.stop {\"id\": \"$PROCESS_ID\"}" | ./bin/compass --cli

sleep 1

echo -e "\n\nFinal log check..."
echo "compass.process.logs {\"id\": \"$PROCESS_ID\", \"limit\": 20}" | ./bin/compass --cli | grep -v "compass>"