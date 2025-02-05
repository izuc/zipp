#!/bin/bash
#
# Builds ZIPP with the latest git tag and commit hash (short)
# E.g.: ./zipp -v --> ZIPP 0.3.0-f3b76ae4

latest_tag=$(git describe --tags $(git rev-list --tags --max-count=1))
commit_hash=$(git rev-parse --short HEAD)

go build -ldflags="-s -w -X github.com/izuc/zipp/plugins/banner.AppVersion=${latest_tag:1}-$commit_hash" -tags badger
