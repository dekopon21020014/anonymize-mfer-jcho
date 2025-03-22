#!/bin/bash
docker build -t anonymize-mfer-jcho:amd64 .
docker save -o anonymize-mfer-jcho.tar anonymize-mfer-jcho:amd64
