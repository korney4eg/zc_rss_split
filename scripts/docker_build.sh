#!/bin/bash
set -e

source .env
docker build . -t korney4eg/zc_feed_parser:${VERSION}
