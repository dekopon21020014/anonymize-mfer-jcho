#!/bin/bash
docker build --platform=linux/amd64 -f ./Dockerfile -t anonymize-mfer:amd64 ../src
docker save -o anonymize-mfer.tar anonymize-mfer:amd64
