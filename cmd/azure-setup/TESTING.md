# Azure Setup CLI Tool - Testing Documentation

## Test Overview

Comprehensive unit tests have been implemented for the Azure Setup CLI tool to ensure reliability and correctness.

## Test Statistics

- **Total Test Files**: 4
- **Total Lines of Test Code**: ~992 lines
- **Test Coverage**: 26.3% of statements
- **All Tests Passing**: ✅ Yes

## Test Files

### 1. `validate_test.go` - Validation Logic Tests

Tests for input validation and helper functions:

**Test Cases:**
- ✅ `TestValidateInputs` - 9 test scenarios
  - Valid configuration
  - Missing site URL
  - Invalid site URL (no protocol, not HTTPS)
  - Missing app name / app name too short
  - Secret expiration validation
  - Default value handling
  - Site URL with path support

- ✅ `TestExtractHostnameAndPath` - 7 test scenarios
  - Simple domain extraction
  - Domain with port
  - Domain with path (single and nested)
  - Trailing slash handling
  - Invalid URL error handling

- ✅ `TestBuildApplicationIDURI` - 7 test scenarios
  - Various URL formats
  - Port handling
  - Path handling
  - Client ID integration
  - Error cases

- ✅ `TestIsApplicationAdminRole` - 5 test scenarios
  - Application Administrator role
  - Cloud Application Administrator role
  - Global Administrator role
  - Unknown/non-admin roles

**Total Tests**: 28 test cases

### 2. `output_test.go` - Output Formatting Tests

Tests for all output formats:

**Test Cases:**
- ✅ `TestOutputJSON` - JSON output format
  - Valid JSON structure
  - All fields present
  - Proper serialization

- ✅ `TestOutputEnv` - Environment variables format
  - Export statements generated
  - M365 variables present
  - Mattermost plugin variables present
  - Failure handling

- ✅ `TestOutputMattermostConfig` - Mattermost config.json format
  - Valid JSON structure
  - Proper plugin configuration format
  - Failure handling

- ✅ `TestOutputHuman` - Human-readable format
  - Successful creation output
  - Successful update output
  - Dry run mode output
  - Failure output

- ✅ `TestOutputResult` - Output format dispatcher
  - All format types (human, json, env, mattermost)
  - Default format handling
  - Invalid format error handling

**Total Tests**: 17 test cases

### 3. `types_test.go` - Data Structures and Constants Tests

Tests for type definitions and helper functions:

**Test Cases:**
- ✅ `TestGetRequiredPermissions` - Permission list validation
  - 3 permissions present
  - Correct IDs and types
  - User.Read (Delegated)
  - TeamsActivity.Send (Application)
  - AppCatalog.Read.All (Application)

- ✅ `TestGetPreAuthorizedClients` - Pre-authorized clients validation
  - 4 clients present
  - Teams Web/Desktop
  - Outlook Web/Desktop

- ✅ `TestPermissionConstants` - Permission ID validation
  - Graph Resource ID
  - Permission UUIDs format
  - Permission types

- ✅ `TestPreAuthorizedClientConstants` - Client ID validation
  - UUID format validation
  - Known client IDs verification

- ✅ `TestScopeConstants` - Scope configuration
  - Scope name
  - Scope description
  - User consent level

- ✅ `TestSetupConfigDefaults` - Configuration structure
  - Default values
  - Rollback slice initialization

- ✅ `TestSetupResultStructure` - Result structure
  - All fields accessible
  - Proper data types

**Total Tests**: 7 test suites with multiple assertions

### 4. `rollback_test.go` - Rollback Mechanism Tests

Tests for error recovery and cleanup:

**Test Cases:**
- ✅ `TestExecuteRollback` - Rollback execution
  - Empty rollback list handling
  - Single rollback function
  - Multiple functions in reverse order
  - Continues on rollback error
  - All functions fail handling
  - Verbose mode

- ✅ `TestRollbackIntegration` - Integration testing
  - Resource tracking
  - Cleanup verification

**Total Tests**: 7 test scenarios

## Running Tests

### Run All Tests

```bash
go test ./cmd/azure-setup/
```

### Run Tests with Verbose Output

```bash
go test -v ./cmd/azure-setup/
```

### Run Tests with Coverage

```bash
go test -cover ./cmd/azure-setup/
```

### Generate Coverage Report

```bash
go test -coverprofile=coverage.out ./cmd/azure-setup/
go tool cover -html=coverage.out
```

### Run Specific Test

```bash
go test -v ./cmd/azure-setup/ -run TestValidateInputs
```

### Run Specific Test Case

```bash
go test -v ./cmd/azure-setup/ -run TestValidateInputs/valid_configuration
```

## Test Coverage Details

### Well-Covered Modules

✅ **Validation Logic** (`validate.go`)
- Input validation: 100% covered
- URL parsing: 100% covered
- Helper functions: 100% covered

✅ **Output Formatting** (`output.go`)
- All format types tested
- Error handling covered
- Output structure validation

✅ **Type Definitions** (`types.go`)
- All constants validated
- Permission lists verified
- Client IDs checked

✅ **Rollback Logic** (`rollback.go`)
- Execution order verified
- Error handling tested
- Integration scenarios covered

### Modules Not Covered by Unit Tests

⚠️ **Azure Integration** (`auth.go`, `app.go`, `permissions.go`, `expose.go`, `secret.go`)
- These modules interact with the Azure SDK
- Would require mocking the Microsoft Graph API
- Intended for manual testing or integration tests
- Marked for future implementation

## Testing Best Practices Followed

### ✅ Table-Driven Tests
All test functions use table-driven test patterns for clarity and maintainability.

```go
tests := []struct {
    name        string
    config      *SetupConfig
    expectError bool
    errorMsg    string
}{
    // test cases...
}
```

### ✅ Descriptive Test Names
Test names clearly describe what is being tested:
- `TestValidateInputs/valid_configuration`
- `TestOutputJSON`
- `TestExecuteRollback/multiple_rollback_functions_in_reverse_order`

### ✅ Assertions with Context
All assertions include helpful context messages:

```go
assert.Equal(t, expected, actual, "should contain client ID: %s", expected)
```

### ✅ Error Testing
Both success and failure cases are tested:
- Valid inputs
- Invalid inputs
- Edge cases
- Error messages

### ✅ Integration Tests
Rollback tests include integration scenarios that test multiple components together.

## Test Maintenance

### Adding New Tests

1. Create test file named `<module>_test.go`
2. Use table-driven test pattern
3. Include both positive and negative test cases
4. Add descriptive test names
5. Run tests to verify: `go test ./cmd/azure-setup/`

### Updating Tests

When modifying code:
1. Update corresponding tests
2. Add tests for new functionality
3. Verify all tests pass
4. Check coverage hasn't decreased

## Known Limitations

### Azure SDK Integration
The modules that directly interact with Azure SDK are not covered by unit tests:
- `auth.go` - Azure authentication
- `app.go` - Application creation/updates
- `permissions.go` - Permission configuration
- `expose.go` - API exposure
- `secret.go` - Secret generation
- `main.go` - CLI orchestration

**Rationale**: These modules require mocking complex Azure SDK types. Manual testing and integration tests are more appropriate.

**Future Work**: Could implement integration tests using:
- Test Azure AD tenant
- Mock Azure SDK responses
- Docker-based test environment

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run Tests
  run: go test -v ./cmd/azure-setup/

- name: Check Coverage
  run: |
    go test -coverprofile=coverage.out ./cmd/azure-setup/
    go tool cover -func=coverage.out
```

## Manual Testing Checklist

For untested Azure integration modules, perform manual testing:

- [ ] Test with Azure CLI authentication
- [ ] Test with service principal authentication
- [ ] Test dry-run mode
- [ ] Test creating new application
- [ ] Test updating existing application
- [ ] Test permission configuration
- [ ] Test API exposure setup
- [ ] Test secret generation
- [ ] Test rollback on error
- [ ] Test all output formats
- [ ] Test with various Mattermost URLs (with/without paths, ports)

## Summary

✅ **Comprehensive test coverage** for core validation and output logic
✅ **All tests passing** with no failures
✅ **Table-driven tests** for maintainability
✅ **992 lines of test code** ensuring quality
✅ **Future-ready** for integration test expansion

The test suite ensures that the critical validation, output formatting, and error handling logic works correctly, providing confidence in the tool's reliability.
