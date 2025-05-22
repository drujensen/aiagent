#!/bin/bash

# Ensure the script runs as ubuntu user
if [ "$(whoami)" != "ubuntu" ]; then
    echo "This script must be run as the ubuntu user. Use 'sudo -u ubuntu $0'"
    exit 1
fi

# Define the executable name
EXECUTABLE=/usr/local/bin/aiagent

# Check if the process is running
if ! pgrep -f "$EXECUTABLE" > /dev/null; then
    echo "No running process found for $EXECUTABLE"
    exit 1
fi

# Stop the process
pkill -f "$EXECUTABLE"

# Verify the process stopped
sleep 1
if pgrep -f "$EXECUTABLE" > /dev/null; then
    echo "Failed to stop $EXECUTABLE"
    exit 1
else
    echo "Stopped $EXECUTABLE successfully"
fi
