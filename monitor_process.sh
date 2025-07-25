#!/bin/bash

PROCESS_ID="c830fc36"

echo "Starting process..."
echo "compass.process.start {\"id\": \"$PROCESS_ID\"}" | ./bin/compass --cli 2>&1

sleep 5

echo -e "\n\nChecking logs after 5 seconds..."
echo "compass.process.logs {\"id\": \"$PROCESS_ID\", \"limit\": 10}" | ./bin/compass --cli 2>&1

sleep 5

echo -e "\n\nChecking logs after 10 seconds..."
echo "compass.process.logs {\"id\": \"$PROCESS_ID\", \"limit\": 10}" | ./bin/compass --cli 2>&1

echo -e "\n\nStopping process..."
echo "compass.process.stop {\"id\": \"$PROCESS_ID\"}" | ./bin/compass --cli 2>&1

sleep 2

echo -e "\n\nFinal log check..."
echo "compass.process.logs {\"id\": \"$PROCESS_ID\", \"limit\": 20}" | ./bin/compass --cli 2>&1