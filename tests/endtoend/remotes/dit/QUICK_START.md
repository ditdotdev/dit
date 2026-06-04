# Quick Start: Running Auth E2E Tests

This guide gets you running the auth E2E tests in under 5 minutes.

## Prerequisites Check

```bash
# Check if Docker is running
docker ps

# Check if dit CLI is built
which dit || (cd dit && make build)
```

## One-Command Setup

```bash
# Start all services
cd dit-remote-server/deploy/compose
docker-compose up -d

# Wait for services to be ready (30 seconds)
sleep 30
```

## Run Tests

```bash
# Run auth E2E tests
cd dit
make test-auth-workflow
```

## Verify Results

✅ **Success**: All tests pass with green checkmarks  
❌ **Failure**: Check the error output and see troubleshooting below

## Quick Troubleshooting

### Services not healthy?
```bash
# Check service status
docker ps | grep dit

# Check logs
docker logs dit-auth-server
docker logs dit-api-gateway
```

### Database connection issues?
```bash
# Verify PostgreSQL is running
docker exec dit-postgres pg_isready -U dit

# Check database logs
docker logs dit-postgres
```

### Test data cleanup?
```bash
# Manually cleanup test users
docker exec dit-postgres psql -U dit -d dit -c "
DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login LIKE 'test%');
DELETE FROM whitelisted_users WHERE github_login LIKE 'test%';
DELETE FROM access_requests WHERE github_login LIKE 'test%';
DELETE FROM users WHERE github_login LIKE 'test%';
"
```

## Full Reset

If things are really broken:

```bash
# Stop everything
cd dit-remote-server/deploy/compose
docker-compose down -v

# Restart from scratch
docker-compose up -d
sleep 30

# Run tests again
cd ../../dit
make test-auth-workflow
```

## Test Breakdown

The test suite runs in this order:

1. **Health Checks** (3 tests) - Verify services are running
2. **Setup** (4 tests) - Create test users and data
3. **Auth Tests** (35+ tests) - Test authentication flows
4. **Cleanup** (2 tests) - Remove test data

**Total Duration**: ~2-3 minutes

## What Each Test Validates

- ✅ JWT tokens work correctly
- ✅ Unauthenticated users are blocked
- ✅ Authenticated users can access their resources
- ✅ Admins have elevated privileges
- ✅ Access requests can be approved/rejected
- ✅ Whitelist management works
- ✅ Sessions are tracked properly
- ✅ Audit logs record events
- ✅ Rate limiting is enforced

## Environment Variables (Optional)

If you need custom configuration:

```bash
# Auth server port (default: 8085)
export AUTH_PORT=8085

# API gateway port (default: 8080)
export API_PORT=8080

# PostgreSQL port (default: 5433)
export POSTGRES_PORT=5433
```

## Next Steps

✅ **Tests passing?** You're ready to deploy!  
❌ **Tests failing?** Check the full documentation at:
- `tests/endtoend/remotes/ditdotdev/README.md`
- `AUTH_IMPLEMENTATION.md`
- `DEPLOYMENT_GUIDE.md`

## Support

- **GitHub Issues**: Report problems at github.com/ditdotdev/dit
- **Documentation**: See `E2E_TESTING_SUMMARY.md` for full details
- **Logs**: Always check Docker logs first
