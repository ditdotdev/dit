# Dit Remote Server E2E Tests

This directory contains end-to-end tests for the Dit Remote Server.

## Test Files

### ditWorkflowTests.yml
Tests the core dit workflow functionality:
- Repository creation and management
- Commit push/pull operations
- Remote repository operations
- Web UI integration

### authWorkflowTests.yml
Tests authentication and authorization functionality:
- JWT token validation
- User authentication and whitelisting
- Admin access controls
- Access request workflow
- Session management
- Audit logging
- Rate limiting
- Integration with dit CLI

## Prerequisites

Before running the tests, ensure:

1. **dit-remote-server is running**
   ```bash
   cd dit-remote-server/deploy/compose
   docker-compose up -d
   ```

2. **Auth service is configured**
   - `.env.auth` file exists with GitHub OAuth credentials
   - JWT keys are generated (`jwt-private.pem` and `jwt-public.pem`)
   - Database is migrated (Liquibase migrations applied)

3. **dit CLI is built**
   ```bash
   cd dit
   make build
   ```

4. **BATS test runner is installed**
   ```bash
   npm install -g bats
   ```

## Running Tests

### Run all dit workflow tests:
```bash
make test-dit-workflow
```

### Run all auth workflow tests:
```bash
make test-auth-workflow
```

### Run both test suites:
```bash
make test-dit-workflow
make test-auth-workflow
```

### Run specific test file manually:
```bash
bats tests/endtoend/remotes/dit/dit-workflow.bats
bats tests/endtoend/remotes/dit/auth-workflow.bats
```

## Test Coverage

### Authentication Tests
- ✅ JWT token generation and validation
- ✅ Expired token rejection
- ✅ Malformed token rejection
- ✅ Unauthenticated access control
- ✅ Authenticated user access
- ✅ Admin user access
- ✅ Admin-only endpoint protection

### Authorization Tests
- ✅ Whitelist checking
- ✅ Non-whitelisted user blocking
- ✅ Access request creation
- ✅ Access request approval workflow
- ✅ Direct whitelist management

### Session Management
- ✅ Session creation
- ✅ Session revocation
- ✅ Session expiration

### Audit Logging
- ✅ Audit log entry creation
- ✅ Metadata tracking
- ✅ Event type logging

### Rate Limiting
- ✅ Anonymous rate limits (10/s)
- ✅ Authenticated rate limits (50/s)
- ✅ Admin rate limits (100/s)

### Integration Tests
- ✅ Dit CLI with authenticated API
- ✅ Repository creation with auth
- ✅ Commit operations with auth

## Test Architecture

The auth tests use a multi-tier approach:

1. **Database Setup**: Creates test users directly in PostgreSQL
2. **JWT Generation**: Generates valid JWTs using the auth server's private key
3. **API Testing**: Tests HTTP endpoints with various authentication states
4. **Cleanup**: Removes all test data after completion

## Debugging Failed Tests

### Check service health:
```bash
curl http://localhost:8085/health  # Auth server
curl http://localhost:8080/health  # API gateway
docker exec dit-postgres pg_isready  # Database
```

### View service logs:
```bash
docker logs dit-auth-server
docker logs dit-api-gateway
docker logs dit-postgres
```

### Check database state:
```bash
docker exec -it dit-postgres psql -U dit -d dit
```

### Verify JWT configuration:
```bash
docker exec dit-auth-server env | grep JWT
```

### Manual test cleanup:
```bash
docker exec dit-postgres psql -U dit -d dit -c "
DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login LIKE 'test%');
DELETE FROM whitelisted_users WHERE github_login LIKE 'test%';
DELETE FROM access_requests WHERE github_login LIKE 'test%';
DELETE FROM users WHERE github_login LIKE 'test%';
"
```

## CI/CD Integration

These tests are designed to run in CI/CD pipelines:

```yaml
# .github/workflows/e2e-tests.yml
name: E2E Tests

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  e2e-auth-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Start dit-remote-server
        run: |
          cd dit-remote-server/deploy/compose
          docker-compose up -d
          
      - name: Wait for services
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:8085/health; do sleep 2; done'
          timeout 60 bash -c 'until curl -f http://localhost:8080/health; do sleep 2; done'
      
      - name: Run auth E2E tests
        run: make test-auth-workflow
```

## Test Environment

The tests expect the following services to be running:

| Service | Port | Container Name | Health Check |
|---------|------|----------------|--------------|
| Auth Server | 8085 | dit-auth-server | http://localhost:8085/health |
| API Gateway | 8080 | dit-api-gateway | http://localhost:8080/health |
| PostgreSQL | 5433 | dit-postgres | pg_isready |
| MinIO | 9000 | dit-minio | http://localhost:9000/minio/health/live |

## Known Issues and Limitations

1. **OAuth Flow**: Tests use direct JWT generation rather than full GitHub OAuth flow
2. **Rate Limiting**: Some rate limit tests may be flaky under high system load
3. **Test Isolation**: Tests cleanup after themselves but may leave data on failures
4. **Timing**: Some tests may need retries if services are slow to start

## Contributing

When adding new auth tests:

1. Follow the existing test structure
2. Add cleanup steps to remove test data
3. Use descriptive test names
4. Include both positive and negative test cases
5. Update this README with new test coverage

## Support

For issues with tests:
- Check the [DEPLOYMENT_GUIDE.md](../../../../dit-remote-server/DEPLOYMENT_GUIDE.md)
- Review [AUTH_IMPLEMENTATION.md](../../../../dit-remote-server/AUTH_IMPLEMENTATION.md)
- Open an issue in the repository
