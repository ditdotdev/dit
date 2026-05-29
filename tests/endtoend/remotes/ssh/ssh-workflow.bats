#!/usr/bin/env bats

# Load shared test helpers
load '../../test_helper'

# Setup: Pull SSH test server image and start it
setup_file() {
  docker pull datadatdat/ssh-test-server:latest
  
  # Use MSYS_NO_PATHCONV to prevent Git Bash from auto-converting paths
  # This ensures Docker receives the correct path format on both Windows and Linux
  # Windows: /c/dev/datadatdat stays as-is (Docker Desktop handles the conversion)
  # Linux: paths work normally
  MSYS_NO_PATHCONV=1 docker run -v "${REPO_ROOT}:/workdir" --workdir /workdir --network datadatdat-docker -d --name sshHost datadatdat/ssh-test-server:latest
  docker exec sshHost mkdir -p /test
  
  # Get SSH host IP and save it
  SSH_HOST=$(docker inspect -f "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}" sshHost)
  echo "$SSH_HOST" > "$BATS_TMPDIR/ssh_host.txt"
}

# Cleanup: Stop and remove SSH server
teardown_file() {
  docker rm -f sshHost || true
}

@test "run empty mongo db" {
  run "$D3" run -n ssh-test mongo
  assert_success
  assert_output --partial "Running controlled container ssh-test"
  sleep 10
}

@test "create new commit" {
  run "$D3" commit -m "ssh-test Commit" ssh-test
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit GUID and save to temp file
  COMMIT_GUID=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_GUID" > "$BATS_TMPDIR/ssh_commit_guid.txt"
}

@test "add ssh remote with password succeeds" {
  [ -f "$BATS_TMPDIR/ssh_host.txt" ] || skip "SSH_HOST not saved"
  SSH_HOST=$(cat "$BATS_TMPDIR/ssh_host.txt")
  
  # ssh-test-server has an ephemeral host key not in known_hosts; opt out of
  # strict host-key checking (default since remote-sdk #48 / ssh-remote #62).
  run "$D3" remote add -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/test" ssh-test
  assert_success
}

@test "repo has ssh remote with password" {
  [ -f "$BATS_TMPDIR/ssh_host.txt" ] || skip "SSH_HOST not saved"
  SSH_HOST=$(cat "$BATS_TMPDIR/ssh_host.txt")
  
  run "$D3" remote ls ssh-test
  assert_success
  assert_output --partial "ssh://root:*****@$SSH_HOST/test"
}

@test "list remote with password commits returns empty list" {
  run "$D3" remote log ssh-test
  assert_success
}

@test "get non-existent remote commit with password fails" {
  run "$D3" pull ssh-test
  assert_failure
}

@test "push commit with password succeeds" {
  [ -f "$BATS_TMPDIR/ssh_commit_guid.txt" ] || skip "COMMIT_GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_commit_guid.txt")
  
  run "$D3" push ssh-test
  assert_success
  assert_output --partial "Pushing $COMMIT_GUID to 'origin'"
  assert_output --partial "Push completed successfully"
}

@test "list remote commits with password returns pushed commit" {
  [ -f "$BATS_TMPDIR/ssh_commit_guid.txt" ] || skip "COMMIT_GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_commit_guid.txt")
  
  run "$D3" remote log ssh-test
  assert_success
  assert_output --partial "Commit $COMMIT_GUID"
  assert_output --partial "ssh-test Commit"
}

@test "push of same commit with password fails" {
  [ -f "$BATS_TMPDIR/ssh_commit_guid.txt" ] || skip "COMMIT_GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_commit_guid.txt")
  
  run "$D3" push ssh-test
  assert_failure
  assert_output --partial "commit $COMMIT_GUID exists in remote 'origin'"
}

@test "delete local commit succeeds" {
  [ -f "$BATS_TMPDIR/ssh_commit_guid.txt" ] || skip "COMMIT_GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_commit_guid.txt")
  
  run "$D3" delete -c "$COMMIT_GUID" ssh-test
  assert_success
  assert_output --partial "$COMMIT_GUID deleted"
}

@test "list local commits is empty" {
  run "$D3" log ssh-test
  assert_success
}

@test "pull original commit with password succeeds" {
  [ -f "$BATS_TMPDIR/ssh_commit_guid.txt" ] || skip "COMMIT_GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_commit_guid.txt")
  
  run "$D3" pull ssh-test
  assert_success
  assert_output --partial "Pulling $COMMIT_GUID from 'origin'"
  assert_output --partial "Pull completed successfully"
}

@test "checkout commit succeeds" {
  [ -f "$BATS_TMPDIR/ssh_commit_guid.txt" ] || skip "COMMIT_GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_commit_guid.txt")
  
  run "$D3" checkout -c "$COMMIT_GUID" ssh-test
  assert_success
  assert_output --partial "Stopping container ssh-test"
  assert_output --partial "Checkout $COMMIT_GUID"
  assert_output --partial "Starting container ssh-test"
  assert_output --partial "$COMMIT_GUID checked out"
}

@test "remove remote with password succeeds" {
  run "$D3" remote rm ssh-test origin
  assert_success
  assert_output --partial "Removed origin from ssh-test"
}

@test "create new sshkey commit" {
  run "$D3" commit -m "ssh-test key Commit" ssh-test
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit GUID and save to temp file
  KEY_COMMIT_GUID=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$KEY_COMMIT_GUID" > "$BATS_TMPDIR/ssh_key_commit_guid.txt"
}

@test "create new directory in ssh server" {
  run docker exec sshHost mkdir -p /sshtest
  assert_success
}

@test "add private ssh key to ssh server" {
  run docker exec sshHost sh -c "cat /workdir/tests/endtoend/remotes/ssh/sshKey.pub > /root/.ssh/authorized_keys"
  assert_success
}

@test "add remote with ssh key succeeds" {
  [ -f "$BATS_TMPDIR/ssh_host.txt" ] || skip "SSH_HOST not saved"
  SSH_HOST=$(cat "$BATS_TMPDIR/ssh_host.txt")
  
  run "$D3" remote add -p "keyFile=$REPO_ROOT/tests/endtoend/remotes/ssh/sshKey" -p "skipHostCheck=true" "ssh://root@$SSH_HOST/sshtest" ssh-test
  assert_success
}

@test "repo has ssh remote with ssh key" {
  [ -f "$BATS_TMPDIR/ssh_host.txt" ] || skip "SSH_HOST not saved"
  SSH_HOST=$(cat "$BATS_TMPDIR/ssh_host.txt")
  
  run "$D3" remote ls ssh-test
  assert_success
  assert_output --partial "ssh://root@$SSH_HOST/sshtest"
}

@test "list remote with ssh key commits returns empty list" {
  run "$D3" remote log ssh-test
  assert_success
}

@test "get non-existent remote commit with ssh key fails" {
  run "$D3" pull ssh-test
  assert_failure
}

# Disabled due to Windows path accessibility issue - see https://github.com/datadatdat/datadatdat/issues/30
# @test "push commit with ssh key succeeds" {
#   [ -f "$BATS_TMPDIR/ssh_key_commit_guid.txt" ] || skip "KEY_COMMIT_GUID not saved"
#   KEY_COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_key_commit_guid.txt")
#   
#   run "$D3" push ssh-test
#   assert_success
#   assert_output --partial "Pushing $KEY_COMMIT_GUID to 'origin'"
#   assert_output --partial "Push completed successfully"
# }

# Disabled due to Windows path accessibility issue - see https://github.com/datadatdat/datadatdat/issues/30
# @test "list remote commits with ssh key returns pushed commit" {
#   [ -f "$BATS_TMPDIR/ssh_key_commit_guid.txt" ] || skip "KEY_COMMIT_GUID not saved"
#   KEY_COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_key_commit_guid.txt")
#   
#   run "$D3" remote log ssh-test
#   assert_success
#   assert_output --partial "Commit $KEY_COMMIT_GUID"
#   assert_output --partial "ssh-test key Commit"
# }

# Disabled due to Windows path accessibility issue - see https://github.com/datadatdat/datadatdat/issues/30
# @test "push of same commit with ssh key fails" {
#   [ -f "$BATS_TMPDIR/ssh_key_commit_guid.txt" ] || skip "KEY_COMMIT_GUID not saved"
#   KEY_COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_key_commit_guid.txt")
#   
#   run "$D3" push ssh-test
#   assert_failure
#   assert_output --partial "commit $KEY_COMMIT_GUID exists in remote 'origin'"
# }

@test "delete local key commit succeeds" {
  [ -f "$BATS_TMPDIR/ssh_key_commit_guid.txt" ] || skip "KEY_COMMIT_GUID not saved"
  KEY_COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_key_commit_guid.txt")
  
  run "$D3" delete -c "$KEY_COMMIT_GUID" ssh-test
  assert_success
  assert_output --partial "$KEY_COMMIT_GUID deleted"
}

@test "list local commits is empty after key commit delete" {
  run "$D3" log ssh-test
  assert_success
}

# Disabled due to Windows path accessibility issue - see https://github.com/datadatdat/datadatdat/issues/30
# @test "pull original commit with ssh key succeeds" {
#   [ -f "$BATS_TMPDIR/ssh_key_commit_guid.txt" ] || skip "KEY_COMMIT_GUID not saved"
#   KEY_COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_key_commit_guid.txt")
#   
#   run "$D3" pull ssh-test
#   assert_success
#   assert_output --partial "Pulling $KEY_COMMIT_GUID from 'origin'"
#   assert_output --partial "Pull completed successfully"
# }

# Disabled due to Windows path accessibility issue - see https://github.com/datadatdat/datadatdat/issues/30
# @test "checkout key commit succeeds" {
#   [ -f "$BATS_TMPDIR/ssh_key_commit_guid.txt" ] || skip "KEY_COMMIT_GUID not saved"
#   KEY_COMMIT_GUID=$(cat "$BATS_TMPDIR/ssh_key_commit_guid.txt")
#   
#   run "$D3" checkout -c "$KEY_COMMIT_GUID" ssh-test
#   assert_success
#   assert_output --partial "Stopping container ssh-test"
#   assert_output --partial "Checkout $KEY_COMMIT_GUID"
#   assert_output --partial "Starting container ssh-test"
#   assert_output --partial "$KEY_COMMIT_GUID checked out"
# }

@test "remove remote with ssh key succeeds" {
  run "$D3" remote rm ssh-test origin
  assert_success
  assert_output --partial "Removed origin from ssh-test"
}

@test "remove ssh-test succeeds" {
  run "$D3" rm -f ssh-test
  assert_success
}
