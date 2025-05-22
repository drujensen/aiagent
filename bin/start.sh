#!/bin/bash

# Ensure the script runs as ubuntu user
if [ "$(whoami)" != "ubuntu" ]; then
    echo "This script must be run as the ubuntu user. Use 'sudo -u ubuntu $0'"
    exit 1
fi

# Define paths
EXECUTABLE=/usr/local/bin/aiagent
LOGFILE=./output.log

# Check if the executable exists
if [ ! -x "$EXECUTABLE" ]; then
    echo "Executable $EXECUTABLE not found or not executable"
    exit 1
fi

# Check if the process is already running
if pgrep -f "$EXECUTABLE" > /dev/null; then
    echo "Process is already running"
    exit 1
fi

# Start the executable in the background
nohup "$EXECUTABLE" serve -storage mongo > "$LOGFILE" 2>&1 &

# Verify the process started
sleep 1
if pgrep -f "$EXECUTABLE" > /dev/null; then
    echo "Started $EXECUTABLE successfully. Logs are in $LOGFILE"
else
    echo "Failed to start $EXECUTABLE. Check $LOGFILE for errors"
    exit 1
fi
