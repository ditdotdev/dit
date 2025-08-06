#!/usr/bin/env bash
#
# Custom ZFS compatibility script for Docker Desktop 2.1.0.5 with ZFS 0.8.2
#

# Modified minimum ZFS version for legacy compatibility
min_zfs_version=0.8.0

# Return the tag in the ZFS repository we should be using to build ZFS binaries.
function get_zfs_build_version() {
  echo "0.8.2"
}

# Modified version compatibility for ZFS 0.8.x series
function zfs_version_compatible() {
  [[ -z "$1" ]] && return 1
  local req_version=${1%-*}                       # Trim any trailing "-XYZ" modifier
  local req_components=(${req_version//./ })      # Replace periods with spaces
  
  # For ZFS 0.8.x series, allow any 0.8.x version
  if [[ ${req_components[0]} -eq 0 && ${req_components[1]} -eq 8 ]]; then
    return 0
  fi
  
  # Original compatibility logic for other versions
  local min_components=(${min_zfs_version//./ })
  [[ ${min_components[0]} -ne ${req_components[0]} ]] && return 1
  [[ ${min_components[1]} -gt ${req_components[1]} ]] && return 1
  return 0
}

# Rest of the functions remain the same...
function zfs_version_matches() {
  local req_version=${1%-*}
  local build_version=$(get_zfs_build_version)
  [[ $req_version = $build_version ]]
}

function get_asset_url() {
  local asset_name=$1
  echo "https://download.titan-data.io/zfs-releases/$asset_name"
}

function is_zfs_loaded() {
  lsmod | grep "^zfs " >/dev/null 2>&1
}

function get_running_zfs_version() {
  cat /sys/module/zfs/version 2>/dev/null
}

function get_filesystem_zfs_version() {
  local directory=$1
  depmod -b $directory >/dev/null 2>&1
  modinfo -F version -b $directory zfs 2>/dev/null
}

function load_zfs_module() {
  local directory=$1
  depmod -b $directory >/dev/null 2>&1
  modprobe -d $directory zfs >/dev/null 2>&1
}

function get_precompiled_module_url() {
  get_asset_url zfs-$(get_zfs_build_version)-$(uname -r).tar.gz
}

function extract_precompiled_module() {
  local asset_url=$1
  local dstdir=$2
  curl -fsSL $asset_url > $dstdir/zfs.tar.gz || return 1
  cd $dstdir && tar -xzf zfs.tar.gz || return 1
  rm $dstdir/zfs.tar.gz
  return 0
}

function check_zfs_device() {
  if [[ ! -e /dev/zfs ]]; then
      mknod -m 660 /dev/zfs c $(cat /sys/class/misc/zfs/dev |sed 's/:/ /g') >/dev/null 2>&1
  fi
}

function sanity_check_zfs() {
  zpool list >/dev/null 2>&1 || return 1
  zfs list >/dev/null 2>&1 || return 1
  return 0
}

function pool_exists() {
  local pool=$1
  zpool status $pool >/dev/null 2>&1
}

function import_pool() {
  local cachefile=$1
  local pool=$2
  zpool import -f -c $cachefile $pool >/dev/null
}

function create_pool() {
  local pool=$1
  local data=$2
  local mountpoint=$3
  local cachefile=$4
  zpool create -m $mountpoint -o cachefile=$cachefile $pool $data
  zfs create -o mountpoint=legacy -o compression=lz4 $pool/data
  zfs create -o mountpoint=legacy $pool/db
}

function update_pool() {
  local pool=$1
  zfs list $pool/deathrow > /dev/null 2>&1 && zfs destroy $pool/deathrow
  zfs list $pool/repo > /dev/null 2>&1 && zfs destroy -R $pool/repo
  zfs list $pool/data > /dev/null 2>&1 || zfs create -o mountpoint=legacy $pool/data
  zfs list $pool/db > /dev/null 2>&1 || zfs create -o mountpoint=legacy $pool/db
}

function destroy_pool() {
  local pool=$1
  zpool destroy $pool
}

function check_running_zfs() {
  log_start "Checking if compatible ZFS is running"
  local retval=1
  if is_zfs_loaded; then
    local version=$(get_running_zfs_version)
    if ! zfs_version_compatible $version; then
      log_error "System is running ZFS $version incompatible with $(get_zfs_build_version), upgrade and retry"
    fi
    echo "System is running ZFS version $version"
    retval=0
  else
    echo "ZFS is not currently loaded"
  fi
  log_end
  return $retval
}

function load_zfs() {
  local module_dir=$1
  local module_type=$2
  local install_dir=$3
  local retval=1
  log_start "Checking if compatible $module_type ZFS is available"
  version=$(get_filesystem_zfs_version $module_dir)
  if [[ $module_type = "compiled" ]]; then
    zfs_version_matches $version
  else
    zfs_version_compatible $version
  fi
  if [[ $? -eq 0 ]]; then
    echo "Version $version compatible"
    if load_zfs_module $module_dir; then
      echo "ZFS loaded"
      echo $module_dir > $install_dir/installed_zfs
      retval=0
    else
      echo "Failed to load module"
    fi
  else
    if [[ -z "$version" ]]; then
      echo "No ZFS module found"
    else
      echo "Version $version incompatible with $(get_zfs_build_version)"
    fi
  fi
  log_end
  return $retval
}

function load_precompiled_zfs() {
  local dstdir=$1
  local install_dir=$2
  local uname=$(uname -r)
  local retval=1
  log_start "Checking if precompiled ZFS is available for '$uname'"
  rm -rf $dstdir || return 1
  mkdir -p $dstdir || return 1
  local asset_url=$(get_precompiled_module_url)
  if extract_precompiled_module $asset_url $dstdir; then
    echo "Version $uname extracted to $dstdir"
    if load_zfs_module $dstdir; then
      echo "Version $uname loaded"
      echo $dstdir > $install_dir/installed_zfs
      retval=0
    else
      echo "Failed to load precompiled module"
    fi
  else
    echo "No ZFS module found"
  fi
  log_end
  return $retval
}

function compile_and_load_zfs() {
  local dstdir=$1
  local install_dir=$2
  log_start "Building ZFS kernel modules (this could take 30 minutes, submit a request for $(uname -r) prebuilt binaries)"
  mkdir -p $dstdir
  docker run --rm -v $dstdir:/build \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -e ZFS_VERSION=zfs-$(get_zfs_build_version) \
    -e ZFS_CONFIG=kernel titandata/zfs-builder:latest || log_error "ZFS build failed"
  log_end
  if ! load_zfs_module $dstdir; then
    log_error "Failed to load compiled modules"
  fi
  echo $dstdir > $install_dir/installed_zfs
}

function check_zfs() {
  check_zfs_device
  sanity_check_zfs || log_error "ZFS not configured properly, contact help"
}

function unload_zfs() {
  local install_dir=$1
  if [[ is_zfs_loaded && -f $install_dir/installed_zfs ]]; then
    local module_location=$(cat $install_dir/installed_zfs)
    modprobe -d "$module_location" -r zfs || return 1
  fi
  return 0
}

function unmount_filesystems() {
  local pool=$1
  local dirs=$(mount -t zfs | grep ^$pool | awk '{print $3}' | sort -r)
  for dir in $dirs; do
     nsenter -m -u -t 1 -n -i umount $dir
  done
}
