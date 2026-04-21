#!/bin/bash

set -euo pipefail

cd /opt/targets
bash hfc.sh

cd /opt/fuzzers
bash hfc.sh

cd /opt/seeds
mv seeds/* /root/hfc/test/
