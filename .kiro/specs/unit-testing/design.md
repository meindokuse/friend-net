# Design Document: Unit Testing for Auth Service

## Overview

This design document defines the comprehensive unit testing architecture for the auth-service Go application. The auth-service implements JWT authentication, refresh token rotation with grace period, and account management using Clean Architecture. This design covers test organization, mock patterns, and test strategies for all layers: domain entities, utility packages, and application services.

### Key Design Decisions

1. **Test Co-location**: Tests placed alongside source files (`*_test.go`) following Go convention
2. **Table-Driven Tests**: Primary pattern for testing multiple scenarios
3. **testify/mock**: Standard approach for mock implementations with verification
4. **Property-Based Testing**: Used for pure functions with deterministic behavior (hashing, token parsing)

## Architecture

### Test Architecture Overview

```
auth-service/internal/
├── domain/entity/
│   ├── account.go
│   ├── account_test.go          # Domain entity tests
│   ├── session.go
│   ├── session_test.go
│   ├── refresh_pair.go
│   └── refresh_pair_test.go
├── pkg/
│   ├── jwt/
│   │   ├── manager.go
│   │   └── manager_test.go      # Utility tests
│   ├── pass/
│   │   ├── pass.go
│   │   └── pass_test.go
│   └── terror/
│       ├── errors.go
│       └── errors_test.go
└── application/service/auth/
    ├── login/
    │   ├── service.go
    │   └── service_test.go      # Application service tests
    ├── refresh/
    │   └── service_test.go
    ├── logout/
    │   └── service_test.go
    ├── register/
    │   └── service_test.go
    └── introspect/
        └── service_test.go
```

### Test Dependencies

```
┌─────────────────────────────────────────────────────────────────┐
│                      Test Technology Stack                       │
├─────────────────────────────────────────────────────────────────┤
│  testing package      │ Standard Go test framework              │
│  testify/assert       │ Fluent assertions                       │
│  testify/mock         │ Mock generation and verification        │
│  testify/suite        │ Optional test suite organization        │
│  golang.org/x/crypto  │ bcrypt for password testing             │
└─────────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Domain Entity Tests

#### Account Entity Tests (`account_test.go`)

**Test File Location**: `internal/domain/entity/account_test.go`

```go
// TestAccount_NewAccount_ValidInput tests account creation with valid parameters
// Validates: Requirements 1.1, 1.2, 1.4
func TestAccount_NewAccount_ValidInput(t *testing.T) {
    tests := []struct {
        name         string
        email        string
        passwordHash string
    }{
        {"standard email", "user@example.com", "hashedpassword123"},
        {"email with subdomain", "user@mail.example.com", "hash"},
        {"plus addressing", "user+tag@example.com", "hash"},
    }
    // Table-driven test implementation
}

// TestAccount_NewAccount_Timestamps tests timestamp initialization
// Validates: Requirements 1.2
func TestAccount_NewAccount_Timestamps(t *testing.T)

// TestAccount_UpdateLastLogin tests last login update
// Validates: Requirements 1.3
func TestAccount_UpdateLastLogin(t *testing.T)

// TestAccount_LastLoginAt_NilInitially tests initial nil state
// Validates: Requirements 1.4
func TestAccount_LastLoginAt_NilInitially(t *testing.T)
```

#### Session Entity Tests (`session_test.go`)

**Test File Location**: `internal/domain/entity/session_test.go`

```go
// TestSession_NewSession tests session creation
// Validates: Requirements 2.1
func TestSession_NewSession(t *testing.T) {
    tests := []struct {
        name          string
        id            string
        accountID     string
        fingerprint   string
        ip            string
        ua            string
        ttl           time.Duration
        wantStatus    SessionStatus
        wantExpiresAt time.Time
    }{
        // Test cases for various session configurations
    }
}

// TestSession_IsIsActive_ActiveNotExpired tests active session check
// Validates: Requirements 2.2
func TestSession_IsActive_ActiveNotExpired(t *testing.T)

// TestSession_IsActive_Revoked tests revoked session
// Validates: Requirements 2.3
func TestSession_IsActive_Revoked(t *testing.T)

// TestSession_IsActive_Expired tests expired session
// Validates: Requirements 2.4
func TestSession_IsActive_Expired(t *testing.T)

// TestSession_Revoke tests session revocation
// Validates: Requirements 2.5
func TestSession_Revoke(t *testing.T)

// TestSession_UpdateLastSeen tests last seen update
// Validates: Requirements 2.6
func TestSession_UpdateLastSeen(t *testing.T)
```

#### RefreshPair Entity Tests (`refresh_pair_test.go`)

**Test File Location**: `internal/domain/entity/refresh_pair_test.go`

```go
// TestRefreshPair_Match_Current tests matching current hash
// Validates: Requirements 3.1
func TestRefreshPair_Match_Current(t *testing.T)

// TestRefreshPair_Match_PrevWithinGrace tests matching prev within grace period
// Validates: Requirements 3.2
func TestRefreshPair_Match_PrevWithinGrace(t *testing.T)

// TestRefreshPair_Match_PrevExpired tests prev expired
// Validates: Requirements 3.3
func TestRefreshPair_Match_PrevExpired(t *testing.T)

// TestRefreshPair_Match_None tests no match
// Validates: Requirements 3.4
func TestRefreshPair_Match_None(t *testing.T)

// TestRefreshPair_Rotate tests rotation
// Validates: Requirements 3.5
func TestRefreshPair_Rotate(t *testing.T)

// TestRefreshPair_SetCurrent tests setting current
// Validates: Requirements 3.6
func TestRefreshPair_SetCurrent(t *testing.T)
```

### 2. Utility Tests

#### JWT Manager Tests (`manager_test.go`)

**Test File Location**: `internal/pkg/jwt/manager_test.go`

```go
// TestNewManager_SecretTooShort tests secret validation
// Validates: Requirements 4.1
func TestNewManager_SecretTooShort(t *testing.T) {
    tests := []struct {
        name          string
        secretKey     string
        refreshSecret string
        wantErr       bool
    }{
        {"secret too short", "short", "validrefreshsecrethere", true},
        {"refresh secret too short", "validsigningkeythatis32chars", "short", true},
        {"both valid", "validsigningkeythatis32chars", "validrefreshsecrethere", false},
    }
}

// TestManager_GenerateAccessToken tests token generation
// Validates: Requirements 4.4
func TestManager_GenerateAccessToken(t *testing.T)

// TestManager_VerifyAccessToken_Valid tests valid token verification
// Validates: Requirements 4.5
func TestManager_VerifyAccessToken_Valid(t *testing.T)

// TestManager_VerifyAccessToken_Expired tests expired token
// Validates: Requirements 4.6
func TestManager_VerifyAccessToken_Expired(t *testing.T)

// TestManager_VerifyAccessToken_InvalidSignature tests invalid signature
// Validates: Requirements 4.7
func TestManager_VerifyAccessToken_InvalidSignature(t *testing.T)

// TestManager_GenerateRefreshToken tests refresh token generation
// Validates: Requirements 4.8
func TestManager_GenerateRefreshToken(t *testing.T)

// TestManager_ParseRefreshToken tests refresh token parsing
// Validates: Requirements 4.9, 4.10
func TestManager_ParseRefreshToken(t *testing.T)

// TestManager_HashRefreshToken_Deterministic tests hash determinism
// Validates: Requirements 4.11
func TestManager_HashRefreshToken_Deterministic(t *testing.T)

// TestManager_HashFingerprint_Deterministic tests fingerprint hash
// Validates: Requirements 4.12
func TestManager_HashFingerprint_Deterministic(t *testing.T)

// TestManager_HashRefreshToken_DifferentInputs tests hash uniqueness
// Validates: Requirements 4.13
func TestManager_HashRefreshToken_DifferentInputs(t *testing.T)
```

#### Password Hasher Tests (`pass_test.go`)

**Test File Location**: `internal/pkg/pass/pass_test.go`

```go
// TestHasher_Hash tests password hashing
// Validates: Requirements 5.1
func TestHasher_Hash(t *testing.T)

// TestHasher_Compare_Matching tests password comparison
// Validates: Requirements 5.2
func TestHasher_Compare_Matching(t *testing.T)

// TestHasher_Compare_NotMatching tests mismatch
// Validates: Requirements 5.3
func TestHasher_Compare_NotMatching(t *testing.T)

// TestHasher_Match_True tests Match returning true
// Validates: Requirements 5.4
func TestHasher_Match_True(t *testing.T)

// TestHasher_Match_False tests Match returning false
// Validates: Requirements 5.5
func TestHasher_Match_False(t *testing.T)

// TestNew_CostValidation tests cost parameter validation
// Validates: Requirements 5.6, 5.7
func TestNew_CostValidation(t *testing.T)
```

#### Typed Error Tests (`errors_test.go`)

**Test File Location**: `internal/pkg/terror/errors_test.go`

```go
// TestNewNotFoundErr tests not found error creation
// Validates: Requirements 6.1
func TestNewNotFoundErr(t *testing.T)

// TestNewConflictErr tests conflict error creation
// Validates: Requirements 6.2
func TestNewConflictErr(t *testing.T)

// TestNewUnauthorizedErr tests unauthorized error creation
// Validates: Requirements 6.3
func TestNewUnauthorizedErr(t *testing.T)

// TestNewBadRequestErr tests bad request error creation
// Validates: Requirements 6.4
func TestNewBadRequestErr(t *testing.T)

// TestNewInternalErr tests internal error creation
// Validates: Requirements 6.5
func TestNewInternalErr(t *testing.T)

// TestNewForbiddenErr tests forbidden error creation
// Validates: Requirements 6.6
func TestNewForbiddenErr(t *testing.T)

// TestError_WithCause tests Error() with cause
// Validates: Requirements 6.7
func TestError_WithCause(t *testing.T)

// TestError_WithoutCause tests Error() without cause
// Validates: Requirements 6.8
func TestError_WithoutCause(t *testing.T)

// TestUnwrap tests error unwrapping
// Validates: Requirements 6.9
func TestUnwrap(t *testing.T)

// TestIsNotFound tests IsNotFound function
// Validates: Requirements 6.10, 6.11
func TestIsNotFound(t *testing.T)

// TestIsConflict tests IsConflict function
// Validates: Requirements 6.12
func TestIsConflict(t *testing.T)

// TestIsUnauthorized tests IsUnauthorized function
// Validates: Requirements 6.13
func TestIsUnauthorized(t *testing.T)
```

### 3. Application Service Tests

#### Mock Interfaces

```go
// MockAccountProvider implements login.AccountProvider
type MockAccountProvider struct {
    mock.Mock
}

func (m *MockAccountProvider) FindByEmail(ctx context.Context, email string) (*entity.Account, error) {
    args := m.Called(ctx, email)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entity.Account), args.Error(1)
}

// MockSessionManager implements session operations for login/refresh/logout
type MockSessionManager struct {
    mock.Mock
}

func (m *MockSessionManager) Create(ctx context.Context, session *entity.Session) error {
    return m.Called(ctx, session).Error(0)
}

func (m *MockSessionManager) Get(ctx context.Context, sessionID string) (*entity.Session, error) {
    args := m.Called(ctx, sessionID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entity.Session), args.Error(1)
}

func (m *MockSessionManager) GetByAccountID(ctx context.Context, accountID string) ([]*entity.Session, error) {
    args := m.Called(ctx, accountID)
    return args.Get(0).([]*entity.Session), args.Error(1)
}

func (m *MockSessionManager) CountByAccountID(ctx context.Context, accountID string) (int64, error) {
    args := m.Called(ctx, accountID)
    return args.Get(0).(int64), args.Error(1)
}

func (m *MockSessionManager) Revoke(ctx context.Context, sessionID, accountID string) error {
    return m.Called(ctx, sessionID, accountID).Error(0)
}

func (m *MockSessionManager) SaveRefreshPair(ctx context.Context, sessionID string, pair *entity.RefreshPair) error {
    return m.Called(ctx, sessionID, pair).Error(0)
}

func (m *MockSessionManager) GetRefreshPair(ctx context.Context, sessionID string) (*entity.RefreshPair, error) {
    args := m.Called(ctx, sessionID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entity.RefreshPair), args.Error(1)
}

func (m *MockSessionManager) UpdateLastSeen(ctx context.Context, sessionID string) error {
    return m.Called(ctx, sessionID).Error(0)
}

func (m *MockSessionManager) RevokeAllByAccountID(ctx context.Context, accountID string) error {
    return m.Called(ctx, accountID).Error(0)
}

func (m *MockSessionManager) BlacklistAccessToken(ctx context.Context, jti string, ttl int64) error {
    return m.Called(ctx, jti, ttl).Error(0)
}

func (m *MockSessionManager) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
    args := m.Called(ctx, jti)
    return args.Bool(0), args.Error(1)
}

// MockAccountCreator implements register.AccountCreator
type MockAccountCreator struct {
    mock.Mock
}

func (m *MockAccountCreator) CreateWithOutbox(ctx context.Context, account *entity.Account, outbox *entity.OutboxEvent) error {
    return m.Called(ctx, account, outbox).Error(0)
}

func (m *MockAccountCreator) FindByEmail(ctx context.Context, email string) (*entity.Account, error) {
    args := m.Called(ctx, email)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entity.Account), args.Error(1)
}

// MockSessionChecker implements introspect.SessionChecker
type MockSessionChecker struct {
    mock.Mock
}

func (m *MockSessionChecker) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
    args := m.Called(ctx, jti)
    return args.Bool(0), args.Error(1)
}
```

#### Login Service Tests (`service_test.go`)

**Test File Location**: `internal/application/service/auth/login/service_test.go`

```go
// TestLogin_Success tests successful login
// Validates: Requirements 7.1, 7.5, 7.6, 7.7
func TestLogin_Success(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockAccounts := NewMockAccountProvider()
    mockSessions := NewMockSessionManager()
    hasher := pass.New(bcrypt.DefaultCost)
    jwtManager, _ := jwt.NewManager(
        "validsigningkeythatis32chars",
        "validrefreshsecrethere",
        "test-issuer",
        time.Hour,
        24*time.Hour,
        time.Hour,
    )
    
    service := login.NewService(mockAccounts, mockSessions, hasher, jwtManager)
    
    // Setup expectations
    mockAccounts.On("FindByEmail", mock.Anything, "user@example.com").
        Return(&entity.Account{...}, nil)
    mockSessions.On("CountByAccountID", mock.Anything, "account-id").
        Return(int64(0), nil)
    mockSessions.On("Create", mock.Anything, mock.Anything).
        Return(nil)
    mockSessions.On("SaveRefreshPair", mock.Anything, mock.Anything, mock.Anything).
        Return(nil)
    
    result, err := service.Login(context.Background(), login.LoginDTO{...})
    
    assert.NoError(t, err)
    assert.NotEmpty(t, result.AccessToken)
    assert.NotEmpty(t, result.RefreshToken)
    mockSessions.AssertExpectations(t)
}

// TestLogin_NonExistentEmail tests login with non-existent email
// Validates: Requirements 7.2
func TestLogin_NonExistentEmail(t *testing.T)

// TestLogin_IncorrectPassword tests login with wrong password
// Validates: Requirements 7.3
func TestLogin_IncorrectPassword(t *testing.T)

// TestLogin_SessionEviction tests session eviction when limit reached
// Validates: Requirements 7.4
func TestLogin_SessionEviction(t *testing.T)
```

#### Refresh Service Tests (`service_test.go`)

**Test File Location**: `internal/application/service/auth/refresh/service_test.go`

```go
// TestRefresh_CurrentToken tests refresh with current token
// Validates: Requirements 8.1, 8.2
func TestRefresh_CurrentToken(t *testing.T)

// TestRefresh_GracePeriod tests refresh with prev token in grace period
// Validates: Requirements 8.3, 8.4
func TestRefresh_GracePeriod(t *testing.T)

// TestRefresh_ReuseAttack tests reuse attack detection
// Validates: Requirements 8.5
func TestRefresh_ReuseAttack(t *testing.T)

// TestRefresh_InvalidTokenFormat tests invalid token format
// Validates: Requirements 8.6
func TestRefresh_InvalidTokenFormat(t *testing.T)

// TestRefresh_RevokedSession tests revoked session
// Validates: Requirements 8.7
func TestRefresh_RevokedSession(t *testing.T)

// TestRefresh_FingerprintMismatch tests fingerprint mismatch
// Validates: Requirements 8.8
func TestRefresh_FingerprintMismatch(t *testing.T)

// TestRefresh_SessionNotFound tests session not found
// Validates: Requirements 8.9
func TestRefresh_SessionNotFound(t *testing.T)
```

#### Logout Service Tests (`service_test.go`)

**Test File Location**: `internal/application/service/auth/logout/service_test.go`

```go
// TestLogout_AccessToken tests logout with access token
// Validates: Requirements 9.1, 9.4
func TestLogout_AccessToken(t *testing.T)

// TestLogout_RefreshToken tests logout with refresh token
// Validates: Requirements 9.2
func TestLogout_RefreshToken(t *testing.T)

// TestLogout_SessionID tests logout with session ID
// Validates: Requirements 9.3
func TestLogout_SessionID(t *testing.T)

// TestLogout_NoTokenOrSessionID tests error when no identifier provided
// Validates: Requirements 9.5
func TestLogout_NoTokenOrSessionID(t *testing.T)

// TestLogoutAll tests logout all sessions
// Validates: Requirements 9.6
func TestLogoutAll(t *testing.T)
```

#### Register Service Tests (`service_test.go`)

**Test File Location**: `internal/application/service/auth/register/service_test.go`

```go
// TestRegister_Success tests successful registration
// Validates: Requirements 10.1, 10.2, 10.4, 10.5
func TestRegister_Success(t *testing.T)

// TestRegister_DuplicateEmail tests duplicate email error
// Validates: Requirements 10.3
func TestRegister_DuplicateEmail(t *testing.T)
```

#### Introspect Service Tests (`service_test.go`)

**Test File Location**: `internal/application/service/auth/introspect/service_test.go`

```go
// TestIntrospect_ValidToken tests introspection of valid token
// Validates: Requirements 11.1, 11.5
func TestIntrospect_ValidToken(t *testing.T)

// TestIntrospect_ExpiredToken tests introspection of expired token
// Validates: Requirements 11.2
func TestIntrospect_ExpiredToken(t *testing.T)

// TestIntrospect_InvalidToken tests introspection of invalid token
// Validates: Requirements 11.3
func TestIntrospect_InvalidToken(t *testing.T)

// TestIntrospect_BlacklistedToken tests introspection of blacklisted token
// Validates: Requirements 11.4
func TestIntrospect_BlacklistedToken(t *testing.T)

// TestIntrospect_BlacklistError tests blacklist check error
// Validates: Requirements 11.6
func TestIntrospect_BlacklistError(t *testing.T)
```

### 4. Test Helpers

**Test Helpers Location**: Test files will include helper functions for common test setup.

```go
// Test fixtures for account creation
func newTestAccount(t *testing.T) *entity.Account {
    account, err := entity.NewAccount("test@example.com", "hashedpassword")
    require.NoError(t, err)
    return account
}

// Test fixtures for session creation
func newTestSession(t *testing.T, accountID string) *entity.Session {
    sessionID := uuid.NewString()
    return entity.NewSession(sessionID, accountID, "fingerprinthash", "127.0.0.1", "test-agent", time.Hour)
}

// Test fixtures for JWT manager
func newTestJWTManager(t *testing.T) *jwt.Manager {
    manager, err := jwt.NewManager(
        "test-signing-key-that-is-32-chars",
        "test-refresh-secret-that-is-32-ch",
        "test-issuer",
        time.Hour,
        24*time.Hour,
        time.Hour,
    )
    require.NoError(t, err)
    return manager
}

// Time helper for testing expiration
func parseTime(s string) time.Time {
    t, _ := time.Parse(time.RFC3339, s)
    return t
}

// Generate test refresh token
func generateTestRefreshToken(t *testing.T, manager *jwt.Manager, sessionID string) string {
    token, err := manager.GenerateRefreshToken(sessionID)
    require.NoError(t, err)
    return token
}
```

## Data Models

### Test Data Structures

```go
// TestCase represents a generic test case for table-driven tests
type TestCase struct {
    Name     string
    Input    interface{}
    Expected interface{}
    WantErr  bool
    ErrType  string
}

// LoginTestCase represents test case for login service
type LoginTestCase struct {
    Name           string
    Email          string
    Password       string
    AccountExists  bool
    PasswordMatch  bool
    SessionCount   int64
    ExpectedErr    error
    ExpectedErrMsg string
}

// RefreshTestCase represents test case for refresh service
type RefreshTestCase struct {
    Name            string
    RefreshToken    string
    SessionActive   bool
    MatchResult     entity.RefreshMatchResult
    FingerprintMatch bool
    ExpectedErr     error
}

// IntrospectTestCase represents test case for introspect service
type IntrospectTestCase struct {
    Name          string
    Token         string
    TokenValid    bool
    TokenExpired  bool
    IsBlacklisted bool
    ExpectedActive bool
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Account Creation State

*For any* valid email and password hash, `NewAccount` SHALL return an Account with:
- A valid UUID as `ID`
- `IsActive` set to `true`
- `LastLoginAt` set to `nil`
- `CreatedAt` equal to `UpdatedAt` (within 1 second tolerance)

**Validates: Requirements 1.1, 1.2, 1.4**

### Property 2: Account Update Last Login

*For any* Account, after calling `UpdateLastLogin()`, `LastLoginAt` SHALL be non-nil and `UpdatedAt` SHALL be updated to the current UTC timestamp.

**Validates: Requirements 1.3**

### Property 3: Session Creation State

*For any* valid session parameters with TTL `t`, `NewSession` SHALL return a Session with:
- `Status` set to `"active"`
- `CreatedAt` equal to `LastSeenAt` (within 1 second tolerance)
- `ExpiresAt` equal to `CreatedAt + t`

**Validates: Requirements 2.1**

### Property 4: Session Active Invariant

*For any* Session, `IsActive()` SHALL return `true` if and only if both conditions hold:
- `Status == "active"`
- `ExpiresAt > now`

**Validates: Requirements 2.2, 2.3, 2.4**

### Property 5: Session Revoke

*For any* Session, after calling `Revoke()`, `Status` SHALL equal `"revoked"`.

**Validates: Requirements 2.5**

### Property 6: Session Update Last Seen

*For any* Session, after calling `UpdateLastSeen()`, `LastSeenAt` SHALL be updated to the current UTC timestamp.

**Validates: Requirements 2.6**

### Property 7: RefreshPair Match Current

*For any* RefreshPair and hash `h` where `h == Current`, `Match(h)` SHALL return `RefreshMatchCurrent`.

**Validates: Requirements 3.1**

### Property 8: RefreshPair Match Grace Period

*For any* RefreshPair with `Prev` set:
- If `now < PrevExpiresAt`, `Match(Prev)` SHALL return `RefreshMatchPrev`
- If `now >= PrevExpiresAt`, `Match(Prev)` SHALL return `RefreshMatchNone`

**Validates: Requirements 3.2, 3.3**

### Property 9: RefreshPair Match None

*For any* RefreshPair and hash `h` where `h != Current` and `h != Prev`, `Match(h)` SHALL return `RefreshMatchNone`.

**Validates: Requirements 3.4**

### Property 10: RefreshPair Rotate

*For any* RefreshPair with `Current = c` and grace period `g`, after calling `Rotate(newHash, g)`:
- `Prev` SHALL equal the previous `Current` (`c`)
- `PrevExpiresAt` SHALL equal `now + g`
- `Current` SHALL equal `newHash`

**Validates: Requirements 3.5**

### Property 11: RefreshPair SetCurrent

*For any* RefreshPair with `Prev = p` and `PrevExpiresAt = t`, after calling `SetCurrent(newHash)`:
- `Current` SHALL equal `newHash`
- `Prev` SHALL remain `p` (unchanged)
- `PrevExpiresAt` SHALL remain `t` (unchanged)

**Validates: Requirements 3.6**

### Property 12: JWT Manager Secret Validation

*For any* secret string `s`, `NewManager` SHALL return an error if:
- `len(s) < 32` for `secretKey` parameter
- `len(s) < 32` for `refreshSecret` parameter

**Validates: Requirements 4.1, 4.2**

### Property 13: JWT Access Token Round-Trip

*For any* valid `sessionID` and `userID`, the sequence `GenerateAccessToken(sessionID, userID)` followed by `VerifyAccessToken(token)` SHALL return `AccessClaims` with:
- `SessionID` equal to the original `sessionID`
- `Subject` equal to the original `userID`

**Validates: Requirements 4.4, 4.5**

### Property 14: JWT Refresh Token Round-Trip

*For any* valid `sessionID`:
- `GenerateRefreshToken(sessionID)` SHALL return a token in format `"{sessionID}.{randomPart}"` where `len(randomPart) >= 32`
- `ParseRefreshToken(token)` SHALL extract the original `sessionID` and `randomPart`

**Validates: Requirements 4.8, 4.9**

### Property 15: Hash Determinism

*For any* input string `s`:
- `HashRefreshToken(s)` called multiple times SHALL return identical output
- `HashFingerprint(s)` called multiple times SHALL return identical output

**Validates: Requirements 4.11, 4.12**

### Property 16: Hash Uniqueness

*For any* two distinct strings `s1` and `s2` where `s1 != s2`:
- `HashRefreshToken(s1)` SHALL NOT equal `HashRefreshToken(s2)`

**Validates: Requirements 4.13**

### Property 17: Password Hash Round-Trip

*For any* password `p`, the sequence `Hash(p)` followed by `Compare(hash, p)` SHALL return `nil` (success).

**Validates: Requirements 5.1, 5.2**

### Property 18: Password Mismatch Detection

*For any* password `p` and different password `p'` where `p != p'`:
- After `hash = Hash(p)`, `Compare(hash, p')` SHALL return an error
- `Match(hash, p')` SHALL return `false`

**Validates: Requirements 5.3, 5.5**

### Property 19: Password Match True

*For any* password `p` and `hash = Hash(p)`, `Match(hash, p)` SHALL return `true`.

**Validates: Requirements 5.4**

### Property 20: Password Hasher Cost Clamping

*For any* cost value `c`:
- If `c < bcrypt.MinCost`, `New(c)` SHALL use `bcrypt.DefaultCost`
- If `c > bcrypt.MaxCost`, `New(c)` SHALL use `bcrypt.DefaultCost`

**Validates: Requirements 5.6, 5.7**

### Property 21: Typed Error Type Field

*For any* error constructor function:
- `NewNotFoundErr(msg, cause)` SHALL return Error with `Type = "NOT_FOUND"`
- `NewConflictErr(msg, cause)` SHALL return Error with `Type = "CONFLICT"`
- `NewUnauthorizedErr(msg, cause)` SHALL return Error with `Type = "UNAUTHORIZED"`
- `NewBadRequestErr(msg, cause)` SHALL return Error with `Type = "BAD_REQUEST"`
- `NewInternalErr(msg, cause)` SHALL return Error with `Type = "INTERNAL"`
- `NewForbiddenErr(msg, cause)` SHALL return Error with `Type = "FORBIDDEN"`

**Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5, 6.6**

### Property 22: Typed Error Format

*For any* Error `e`:
- If `e.Cause != nil`, `e.Error()` SHALL return `"{TYPE}: {message}: {cause}"`
- If `e.Cause == nil`, `e.Error()` SHALL return `"{TYPE}: {message}"`

**Validates: Requirements 6.7, 6.8**

### Property 23: Typed Error Unwrap

*For any* Error `e`, `e.Unwrap()` SHALL return `e.Cause`.

**Validates: Requirements 6.9**

### Property 24: Typed Error Type Checking

*For any* error `err` created by `NewNotFoundErr`, `IsNotFound(err)` SHALL return `true` and all other `Is*` functions SHALL return `false`.

Similarly for other error types with their corresponding `Is*` function.

**Validates: Requirements 6.10, 6.11, 6.12, 6.13**

## Error Handling

### Error Test Categories

1. **Input Validation Errors**: Testing invalid inputs (empty strings, short secrets, malformed tokens)
2. **State Errors**: Testing invalid state transitions (revoked sessions, expired tokens)
3. **Business Logic Errors**: Testing conflict scenarios (duplicate email, reuse attack)
4. **Infrastructure Errors**: Testing storage failures (mock returns errors)

### Error Assertion Patterns

```go
// Asserting terror error types
func assertError(t *testing.T, err error, expectedType string, expectedMessage string) {
    var terr *terror.Error
    require.True(t, errors.As(err, &terr))
    assert.Equal(t, expectedType, terr.Type)
    assert.Contains(t, terr.Message, expectedMessage)
}

// Example usage in test
func TestLogin_InvalidCredentials(t *testing.T) {
    // ...
    err := service.Login(ctx, dto)
    assertError(t, err, "UNAUTHORIZED", "invalid credentials")
}
```

## Testing Strategy

### Dual Testing Approach

This feature uses both property-based testing and example-based unit tests:

1. **Property-Based Tests**: For pure functions and deterministic operations
   - Hash functions (determinism, uniqueness)
   - Token operations (round-trips)
   - Entity state transitions

2. **Example-Based Unit Tests**: For specific scenarios and edge cases
   - Error conditions
   - Business logic branches
   - Mock interactions

### Property-Based Test Configuration

For property-based testing, we use Go's `testing/quick` package:

```go
import "testing/quick"

// Property test for hash determinism
func TestHashRefreshToken_Deterministic_Property(t *testing.T) {
    manager, _ := jwt.NewManager(
        "test-signing-key-that-is-32-chars",
        "test-refresh-secret-that-is-32-ch",
        "test-issuer",
        time.Hour,
        24*time.Hour,
        time.Hour,
    )
    
    property := func(randomPart string) bool {
        hash1 := manager.HashRefreshToken(randomPart)
        hash2 := manager.HashRefreshToken(randomPart)
        return hash1 == hash2
    }
    
    if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
        t.Error(err)
    }
}

// Property test for token round-trip
func TestAccessToken_RoundTrip_Property(t *testing.T) {
    manager, _ := newTestJWTManager(t)
    
    property := func(sessionID, userID string) bool {
        token, err := manager.GenerateAccessToken(sessionID, userID)
        if err != nil {
            return false
        }
        
        claims, err := manager.VerifyAccessToken(token)
        if err != nil {
            return false
        }
        
        return claims.SessionID == sessionID && claims.Subject == userID
    }
    
    if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
        t.Error(err)
    }
}
```

### Test Tags

Each test will be tagged with a comment referencing the design property:

```go
// Feature: unit-testing, Property 1: Account Timestamp Consistency
func TestAccount_NewAccount_Timestamps(t *testing.T) {
    // ...
}

// Feature: unit-testing, Property 8: JWT Token Round-Trip
func TestAccessToken_RoundTrip(t *testing.T) {
    // ...
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/domain/entity/...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test -run TestLogin_Success ./internal/application/service/auth/login/...
```

### Mock Verification Best Practices

```go
// Always verify mock expectations were met
defer mockSessions.AssertExpectations(t)

// Use specific matchers for complex arguments
mockSessions.On("Create", mock.Anything, mock.MatchedBy(func(s *entity.Session) bool {
    return s.AccountID == "expected-account-id" && s.Status == entity.SessionStatusActive
})).Return(nil)

// Assert call counts
mockSessions.AssertNumberOfCalls(t, "Revoke", 1)
```

### Coverage Requirements

| Component | Target Coverage |
|-----------|----------------|
| Domain Entities | 90%+ |
| JWT Manager | 85%+ |
| Password Hasher | 85%+ |
| Typed Errors | 90%+ |
| Application Services | 80%+ |

### Test Organization Best Practices

1. **Table-Driven Tests**: Use for testing multiple input/output combinations
2. **Subtests**: Use `t.Run()` for organizing related test cases
3. **Test Helpers**: Create reusable helpers for common setup
4. **Parallel Tests**: Use `t.Parallel()` for independent tests
5. **Cleanup**: Use `t.Cleanup()` for resource cleanup

### Requirements Traceability Matrix

| Requirement | Test File | Test Function |
|------------|-----------|---------------|
| 1.1-1.4 | account_test.go | TestAccount_* |
| 2.1-2.6 | session_test.go | TestSession_* |
| 3.1-3.6 | refresh_pair_test.go | TestRefreshPair_* |
| 4.1-4.13 | manager_test.go | TestNewManager_*, TestManager_* |
| 5.1-5.7 | pass_test.go | TestHasher_*, TestNew_* |
| 6.1-6.13 | errors_test.go | TestNew*Err, TestError_*, TestIs* |
| 7.1-7.7 | login/service_test.go | TestLogin_* |
| 8.1-8.9 | refresh/service_test.go | TestRefresh_* |
| 9.1-9.6 | logout/service_test.go | TestLogout_* |
| 10.1-10.5 | register/service_test.go | TestRegister_* |
| 11.1-11.6 | introspect/service_test.go | TestIntrospect_* |
| 12.1-12.9 | All test files | File locations |
| 13.1-13.5 | All test files | Using testify |
