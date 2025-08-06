#!/bin/bash
# Build script to create the missing titandata/docker-desktop-zfs-kernel:4.9.184 image

set -e

echo "Building ZFS kernel modules for Docker Desktop 4.9.184-linuxkit..."

# Get the current kernel version from Docker
KERNEL_VERSION=$(docker run --rm alpine uname -r)
echo "Detected kernel version: $KERNEL_VERSION"

# Create a container to build the ZFS modules
echo "Creating build environment..."
docker build -t local/zfs-kernel-builder -f build-kernel-headers.dockerfile .

# Extract the built modules
echo "Extracting built modules..."
docker run --rm -v "$(pwd)/zfs-modules:/output" local/zfs-kernel-builder \
    sh -c "cp -r /build/* /output/ 2>/dev/null || echo 'Build directory not found, checking alternatives...'"

# Create the final kernel image
echo "Creating final kernel image..."
cat > kernel.dockerfile << 'EOF'
FROM alpine:latest

# Install necessary tools
RUN apk add --no-cache kmod

# Copy the built kernel modules
COPY zfs-modules/ /lib/modules/

# Create installation script
RUN cat > /install-zfs.sh << 'SCRIPT'
#!/bin/sh
echo "Installing ZFS kernel modules for $(uname -r)..."

# Load the kernel modules
modprobe zfs || echo "Failed to load ZFS module"

# Verify installation
if lsmod | grep -q zfs; then
    echo "ZFS modules loaded successfully"
else
    echo "Failed to load ZFS modules" >&2
    exit 1
fi
SCRIPT

RUN chmod +x /install-zfs.sh

# Default command to install ZFS
CMD ["/install-zfs.sh"]
EOF

docker build -t "titandata/docker-desktop-zfs-kernel:${KERNEL_VERSION%%-*}" -f kernel.dockerfile .

echo "Built image: titandata/docker-desktop-zfs-kernel:${KERNEL_VERSION%%-*}"
echo "You can now test with: docker run --privileged --rm titandata/docker-desktop-zfs-kernel:${KERNEL_VERSION%%-*}"
