# Used to create titan:latest. Until we update Docker Hub, use this locally to build titan:latest container.

FROM ubuntu:22.04

# Install required packages and ZFS 2.1.x userspace tools
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        zfsutils-linux libzfs4linux zfs-zed \
        curl wget jq docker.io util-linux kmod \
        postgresql postgresql-contrib \
        openjdk-11-jre-headless \
        socat && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    # Create symlink for PostgreSQL compatibility (Titan expects v12, Ubuntu 22.04 has v14)
    ln -sf /usr/lib/postgresql/14 /usr/lib/postgresql/12

# Copy titan binaries and scripts from the original image
COPY --from=titan:latest /titan /titan

# Remove the old zfs.sh file to ensure clean replacement
RUN rm -f /titan/zfs.sh

# Copy the canonical ZFS compatibility script from zfs-builder
COPY --from=titandata/zfs-builder:latest /custom-zfs.sh /titan/zfs.sh

# Make sure the script is executable
RUN chmod +x /titan/zfs.sh
