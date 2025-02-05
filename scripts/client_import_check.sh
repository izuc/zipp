#!/bin/bash

# This script is intended to be run from the root of the repo!
# It checks if any client library file imports explicitly or implicitly a zipp plugin.
FORBIDDEN_IMPORT="zipp/plugins"

cd client
OUTPUT=$(go list -f '{{.ImportPath}}|{{.Imports}}' ./...)
if [[ "$OUTPUT" =~ .*"$FORBIDDEN_IMPORT".* ]]; then
  echo "ERROR: client library imports a zipp plugin"
  exit 1
fi