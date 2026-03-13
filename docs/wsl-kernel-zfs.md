# Recompiling WSL2 Kernel with OpenZFS Support

## 1. Prerequisites (in WSL2)

```bash
sudo apt update && sudo apt install -y \
  build-essential flex bison dwarves libssl-dev libelf-dev \
  bc python3 pahole git autoconf automake libtool gawk \
  alien fakeroot dkms libblkid-dev uuid-dev libudev-dev \
  libssl-dev zlib1g-dev libaio-dev libattr1-dev libffi-dev \
  python3-dev python3-setuptools python3-cffi nasm
```

## 2. Get the WSL2 Kernel Source

Check your current kernel version first:

```bash
uname -r
# e.g., 5.15.167.4-microsoft-standard-WSL2
```

Clone the matching kernel source:

```bash
cd ~
git clone --depth 1 https://github.com/microsoft/WSL2-Linux-Kernel.git \
  -b linux-msft-wsl-$(uname -r | cut -d- -f1)
cd WSL2-Linux-Kernel
```

If the exact tag doesn't match, check available tags with `git ls-remote --tags https://github.com/microsoft/WSL2-Linux-Kernel.git` and pick the closest one.

## 3. Clone OpenZFS

```bash
cd ~
git clone https://github.com/openzfs/zfs.git
cd zfs
git checkout $(git tag -l | grep -E '^zfs-[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1)
```

This checks out the latest stable release tag.

## 4. Configure the Kernel

```bash
cd ~/WSL2-Linux-Kernel

# Start from Microsoft's WSL2 config
cp Microsoft/config-wsl .config

# Enable required kernel options for ZFS
scripts/config --enable CONFIG_MODULES
scripts/config --enable CONFIG_CRYPTO_DEFLATE
scripts/config --enable CONFIG_ZLIB_DEFLATE
scripts/config --enable CONFIG_ZLIB_INFLATE
scripts/config --module CONFIG_BLK_DEV_LOOP
scripts/config --enable CONFIG_EFI_PARTITION

# Update config (accept defaults for any new options)
make olddefconfig
```

## 5. Build the Kernel

```bash
cd ~/WSL2-Linux-Kernel
make -j$(nproc)
make modules -j$(nproc)
sudo make modules_install
```

## 6. Build ZFS Modules Against the New Kernel

```bash
cd ~/zfs
sh autogen.sh

./configure \
  --prefix=/usr \
  --with-linux=$HOME/WSL2-Linux-Kernel \
  --with-linux-obj=$HOME/WSL2-Linux-Kernel

make -j$(nproc)
sudo make install
sudo ldconfig
```

## 7. Install the Custom Kernel

Copy the built kernel image to the Windows side:

```bash
cp ~/WSL2-Linux-Kernel/arch/x86/boot/bzImage /mnt/c/Users/rober/wsl-kernel-zfs
```

Create or edit `C:\Users\rober\.wslconfig`:

```ini
[wsl2]
kernel=C:\\Users\\rober\\wsl-kernel-zfs
```

## 8. Restart WSL2

From PowerShell (not inside WSL):

```powershell
wsl --shutdown
wsl
```

## 9. Load ZFS After Reboot

Every time WSL starts, you need to load the ZFS module:

```bash
sudo modprobe zfs
```

Verify it loaded:

```bash
lsmod | grep zfs
zfs version
zpool version
```

## 10. Make ZFS Load Automatically

```bash
echo "zfs" | sudo tee /etc/modules-load.d/zfs.conf
```

Note: WSL2 doesn't use systemd by default in older versions. If `modules-load.d` doesn't work, add `sudo modprobe zfs` to your `~/.bashrc` or enable systemd in `.wslconfig`:

```ini
[boot]
systemd=true
```

## Troubleshooting

- **Version mismatch errors** (`Invalid module format`): The ZFS modules must be compiled against the exact same kernel you're booting. Rebuild both together.
- **Missing symbols**: Ensure `CONFIG_MODULES=y` is set in kernel config.
- **Pool import issues**: WSL2 doesn't have real block devices by default. You'll need to use file-backed vdevs (`truncate -s 10G /path/to/vdev`) or pass through physical disks via `wsl --mount`.
