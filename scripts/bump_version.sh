#!/bin/bash

source .env

echo "old version: $VERSION"
NEW_VERSION=$(echo $VERSION + 1 | bc)
echo "new version: $NEW_VERSION"
echo "export VERSION=$NEW_VERSION" > .env
