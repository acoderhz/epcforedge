#! /bin/sh

setup_dir=${PWD}

set -e

curl -X GET -i "Content-Type: application/json"  http://localhost:8091/3gpp-traffic-influence/v1/AF_02/subscriptions/11112

exit 0
