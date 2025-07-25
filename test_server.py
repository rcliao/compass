#!/usr/bin/env python3

import time
import sys

print("Server starting up...")
print("Listening on port 8000", flush=True)

for i in range(10):
    print(f"Request {i+1} processed at {time.strftime('%H:%M:%S')}", flush=True)
    if i % 3 == 0:
        print(f"Warning: Heavy load detected at iteration {i+1}", file=sys.stderr, flush=True)
    time.sleep(1)

print("Server shutting down...")