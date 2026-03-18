#!/bin/bash

set -euo pipefail

ISTIO_VERSION="1.29.1"
ISTIO_URL="https://istio.io/downloadIstio"

echo "Fetching Istio ${ISTIO_VERSION} installer..."
TEMP_DIR=$(mktemp -d)
pushd "$TEMP_DIR" > /dev/null
    curl -sL "$ISTIO_URL" | ISTIO_VERSION=${ISTIO_VERSION} sh -
    cd istio-${ISTIO_VERSION}
    export PATH=$PWD/bin:$PATH
popd

echo "Installing Istio ${ISTIO_VERSION} ..."
istioctl install -f testing/gh-actions/istio-cni.yaml -y

# clean up the temporary directory
rm -rf "$TEMP_DIR"