#!/bin/bash
set -euo pipefail

# Bootstrap script for Go Lambda runtime
# This script is called by AWS Lambda custom runtime
exec /var/task/${go_binary}