#! /bin/sh
setup_dir=${PWD}

set -e

curl -X DELETE http://localhost:8080/af/v1/subscriptions/999

exit 0

