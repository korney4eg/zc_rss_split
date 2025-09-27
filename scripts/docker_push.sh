#!/bin/bash
set -e

source .env
echo "pushing korney4eg/zc_feed_parser:${VERSION}" 
docker push korney4eg/zc_feed_parser:${VERSION}
