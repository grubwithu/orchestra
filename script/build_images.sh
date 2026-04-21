#!/bin/bash

set -e

docker build -t hfc-base:latest .

cd test

docker build -t hfc-test:latest -f base.Dockerfile .



