#!/bin/bash
# GoDash Desktop Launcher
# Ensures proper terminal environment for Bubble Tea TUI

# Change to home directory to avoid permission issues
cd "$HOME" || cd /

# Set up proper environment for TUI
export TERM="${TERM:-xterm-256color}"
export COLORTERM="${COLORTERM:-truecolor}"

# Ensure we have a proper TTY
if [ ! -t 0 ] || [ ! -t 1 ]; then
    # No TTY available, try to run in a proper terminal
    exec /usr/bin/godash
else
    # TTY available, run directly
    exec /usr/bin/godash
fi