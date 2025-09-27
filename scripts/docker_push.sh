#!/bin/bash
set -e

source .env
docker push korney4eg/zc_feed_parser:${VERSION}
