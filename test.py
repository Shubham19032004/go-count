#!/usr/bin/env python3
import time

data = []

print("Starting memory consumption...")

try:
    while True:
        # Allocate 1 MB each iteration
        data.append("X" * 1024 * 1024)
        print(f"Allocated: {len(data)} MB")
        time.sleep(0.1)  # slow down so you can see progress
except KeyboardInterrupt:
    print("Stopped by user.")
