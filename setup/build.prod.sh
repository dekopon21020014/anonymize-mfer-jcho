#!/bin/bash
docker build --platform=linux/amd64 -f ./Dockerfile -t anonymize-mfer-jcho:amd64 ../src
docker save -o anonymize-mfer-jcho.tar anonymize-mfer-jcho:amd64
