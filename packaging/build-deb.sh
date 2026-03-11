#!/bin/bash
set -euo pipefail

VERSION="${VERSION:-0.1.0}"
ARCH="${ARCH:-amd64}"
BINARY="${BINARY:-http-pull}"
PKG_NAME="http-pull_${VERSION}_${ARCH}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT

# Directory structure
mkdir -p "$BUILD_DIR/$PKG_NAME/DEBIAN"
mkdir -p "$BUILD_DIR/$PKG_NAME/usr/bin"
mkdir -p "$BUILD_DIR/$PKG_NAME/etc/http-pull"
mkdir -p "$BUILD_DIR/$PKG_NAME/lib/systemd/system"

# Binary
cp "$BINARY" "$BUILD_DIR/$PKG_NAME/usr/bin/http-pull"
chmod 755 "$BUILD_DIR/$PKG_NAME/usr/bin/http-pull"

# Config (marked as conffile so upgrades don't overwrite user edits)
cp "$SCRIPT_DIR/config/config.yaml" "$BUILD_DIR/$PKG_NAME/etc/http-pull/config.yaml"

# Systemd unit
cp "$SCRIPT_DIR/systemd/http-pull.service" "$BUILD_DIR/$PKG_NAME/lib/systemd/system/http-pull.service"

# Control file
cat > "$BUILD_DIR/$PKG_NAME/DEBIAN/control" <<EOF
Package: http-pull
Version: ${VERSION}
Architecture: ${ARCH}
Maintainer: http-pull maintainers
Description: HTTP file puller with configurable hooks
 A lightweight service that periodically downloads files via HTTP(S),
 stores them locally, and runs configurable hooks afterward.
Depends: systemd
Section: net
Priority: optional
EOF

# Mark config as conffile
cat > "$BUILD_DIR/$PKG_NAME/DEBIAN/conffiles" <<EOF
/etc/http-pull/config.yaml
EOF

# Post-install: reload systemd
cat > "$BUILD_DIR/$PKG_NAME/DEBIAN/postinst" <<'EOF'
#!/bin/bash
set -e
systemctl daemon-reload
echo "http-pull installed. Edit /etc/http-pull/config.yaml, then run:"
echo "  systemctl enable --now http-pull"
EOF
chmod 755 "$BUILD_DIR/$PKG_NAME/DEBIAN/postinst"

# Pre-remove: stop service
cat > "$BUILD_DIR/$PKG_NAME/DEBIAN/prerm" <<'EOF'
#!/bin/bash
set -e
if systemctl is-active --quiet http-pull; then
    systemctl stop http-pull
fi
if systemctl is-enabled --quiet http-pull; then
    systemctl disable http-pull
fi
EOF
chmod 755 "$BUILD_DIR/$PKG_NAME/DEBIAN/prerm"

# Post-remove: reload systemd
cat > "$BUILD_DIR/$PKG_NAME/DEBIAN/postrm" <<'EOF'
#!/bin/bash
set -e
systemctl daemon-reload
EOF
chmod 755 "$BUILD_DIR/$PKG_NAME/DEBIAN/postrm"

dpkg-deb --build "$BUILD_DIR/$PKG_NAME"
mv "$BUILD_DIR/$PKG_NAME.deb" .

echo "Built: $PKG_NAME.deb"
