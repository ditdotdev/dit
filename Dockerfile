# Used to create datadatdat:latest. Until we update Docker Hub, use this locally to build datadatdat:latest container.

FROM ubuntu:24.04

# Install required packages and ZFS 2.1.x userspace tools
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        zfsutils-linux libzfs4linux zfs-zed \
        curl wget jq docker.io util-linux kmod \
        postgresql postgresql-contrib \
        openjdk-11-jre-headless \
        socat sshpass openssh-client rsync && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    # Create symlink for PostgreSQL compatibility (Datadatdat expects v12, Ubuntu 22.04 has v14)
    ln -sf /usr/lib/postgresql/14 /usr/lib/postgresql/12

# Copy datadatdat binaries and scripts from the original image
COPY --from=datadatdat/datadatdat:latest /datadatdat /datadatdat

# Remove the old zfs.sh file to ensure clean replacement
RUN rm -f /datadatdat/zfs.sh

# Copy the canonical ZFS compatibility script from zfs-builder
COPY --from=datadatdat/zfs-builder:latest /custom-zfs.sh /datadatdat/zfs.sh

# Make sure the script is executable
RUN chmod +x /datadatdat/zfs.sh
