#!/bin/bash
set -e

INSTALL_DIR="${1:-/usr/local/share/xray}"

echo "Downloading xray geo assets to $INSTALL_DIR..."

mkdir -p "$INSTALL_DIR"

GEOIP_URL="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
GEOSITE_URL="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"

curl -L -o "$INSTALL_DIR/geoip.dat" "$GEOIP_URL"
curl -L -o "$INSTALL_DIR/geosite.dat" "$GEOSITE_URL"

echo "Downloaded:"
ls -lh "$INSTALL_DIR"/*.dat

echo ""
echo "Set environment variable:"
echo "  export XRAY_LOCATION_ASSET=$INSTALL_DIR"
