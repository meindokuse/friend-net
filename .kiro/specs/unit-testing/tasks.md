# Implementation Plan: Unit Testing for Auth Service

## Overview

This implementation plan covers comprehensive unit testing for the auth-service Go application. The tests will cover domain entities, utility packages, and application services using table-driven tests with testify assertions and mocks. Property-based testing will be used for pure functions with deterministic behavior.

## Tasks

- [ ] 1. Setup test infrastructure
  - [ ] 1.1 Add testify dependencies to go.mod
    - Add `github.com/stretchr/testify` to go.mod
    - Run `go mod tidy` to ensure dependencies are resolved
    - _Requirements: 13.1, 13.2, 13.3_
  
  - [ ] 1.2 Create mock implementations for application services
    - Create mock types for AccountProvider, SessionManager, AccountCreator, SessionChecker
    - Place mocks in respective service test files or shared test helpers
    - _Requirements: 13.3, 13.5_

- [ ] 2. Implement Domain Entity Tests
  - [ ] 2.1 Create Account entity tests
    - Write TestAccount_NewAccount_ValidInput for valid account creation
    - Write TestAccount_NewAccount_Timestamps for timestamp initialization
    - Write TestAccount_UpdateLastLogin for last login update
    - Write TestAccount_LastLoginAt_NilInitially for initial nil state
    - Create test helper `newTestAccount(t)` for reusable account fixtures
    - _Requirements: 1.1, 1.2, 1.3, 1.4_
  
  - [ ]* 2.2 Write property tests for Account entity
    - **Property 1: Account Creation State** - verify UUID, IsActive=true, LastLoginAt=nil, timestamps
    - **Property 2: Account Update Last Login** - verify LastLoginAt and UpdatedAt updates
    - **Validates: Requirements 1.1, 1.2, 1.3, 1.4**
  
  - [ ] 2.3 Create Session entity tests
    - Write TestSession_NewSession for session creation with TTL
    - Write TestSession_IsActive_ActiveNotExpired for active session check
    - Write TestSession_IsActive_Revoked for revoked session
    - Write TestSession_IsActive_Expired for expired session
    - Write TestSession_Revoke for session revocation
    - Write TestSession_UpdateLastSeen for last seen update
    - Create test helper `newTestSession(t, accountID)` for reusable session fixtures
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_
  
  - [ ]* 2.4 Write property tests for Session entity
    - **Property 3: Session Creation State** - verify Status="active", timestamps, ExpiresAt
    - **Property 4: Session Active Invariant** - verify IsActive() logic
    - **Property 5: Session Revoke** - verify Status="revoked" after Revoke()
    - **Property 6: Session Update Last Seen** - verify LastSeenAt update
    - **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 2.6**
  
  - [ ] 2.5 Create RefreshPair entity tests
    - Write TestRefreshPair_Match_Current for matching current hash
    - Write TestRefreshPair_Match_PrevWithinGrace for prev within grace period
    - Write TestRefreshPair_Match_PrevExpired for prev expired
    - Write TestRefreshPair_Match_None for no match
    - Write TestRefreshPair_Rotate for rotation
    - Write TestRefreshPair_SetCurrent for setting current
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_
  
  - [ ]* 2.6 Write property tests for RefreshPair entity
    - **Property 7: RefreshPair Match Current** - verify Match returns RefreshMatchCurrent
    - **Property 8: RefreshPair Match Grace Period** - verify prev token grace period logic
    - **Property 9: RefreshPair Match None** - verify no match returns RefreshMatchNone
    - **Property 10: RefreshPair Rotate** - verify rotation state transitions
    - **Property 11: RefreshPair SetCurrent** - verify Current update preserves Prev
    - **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6**

- [ ] 3. Implement Utility Tests
  - [ ] 3.1 Create JWT Manager tests
    - Write TestNewManager_SecretTooShort for secret validation
    - Write TestManager_GenerateAccessToken for token generation
    - Write TestManager_VerifyAccessToken_Valid for valid token verification
    - Write TestManager_VerifyAccessToken_Expired for expired token
    - Write TestManager_VerifyAccessToken_InvalidSignature for invalid signature
    - Write TestManager_GenerateRefreshToken for refresh token generation
    - Write TestManager_ParseRefreshToken for refresh token parsing
    - Write TestManager_HashRefreshToken_Deterministic for hash determinism
    - Write TestManager_HashFingerprint_Deterministic for fingerprint hash
    - Write TestManager_HashRefreshToken_DifferentInputs for hash uniqueness
    - Create test helper `newTestJWTManager(t)` for reusable JWT manager
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7, 4.8, 4.9, 4.10, 4.11, 4.12, 4.13_
  
  - [ ]* 3.2 Write property tests for JWT Manager
    - **Property 12: JWT Manager Secret Validation** - verify secret length validation
    - **Property 13: JWT Access Token Round-Trip** - verify Generate/Verify consistency
    - **Property 14: JWT Refresh Token Round-Trip** - verify token format and parsing
    - **Property 15: Hash Determinism** - verify hash functions return same output
    - **Property 16: Hash Uniqueness** - verify different inputs produce different hashes
    - **Validates: Requirements 4.1, 4.2, 4.4, 4.5, 4.8, 4.9, 4.11, 4.12, 4.13**
  
  - [ ] 3.3 Create Password Hasher tests
    - Write TestHasher_Hash for password hashing
    - Write TestHasher_Compare_Matching for password comparison
    - Write TestHasher_Compare_NotMatching for mismatch
    - Write TestHasher_Match_True for Match returning true
    - Write TestHasher_Match_False for Match returning false
    - Write TestNew_CostValidation for cost parameter validation
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7_
  
  - [ ]* 3.4 Write property tests for Password Hasher
    - **Property 17: Password Hash Round-Trip** - verify Hash then Compare succeeds
    - **Property 18: Password Mismatch Detection** - verify Compare/Match for different passwords
    - **Property 19: Password Match True** - verify Match returns true for correct password
    - **Property 20: Password Hasher Cost Clamping** - verify cost is clamped to valid range
    - **Validates: Requirements 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7**
  
  - [ ] 3.5 Create Typed Error tests
    - Write TestNewNotFoundErr for not found error creation
    - Write TestNewConflictErr for conflict error creation
    - Write TestNewUnauthorizedErr for unauthorized error creation
    - Write TestNewBadRequestErr for bad request error creation
    - Write TestNewInternalErr for internal error creation
    - Write TestNewForbiddenErr for forbidden error creation
    - Write TestError_WithCause for Error() with cause
    - Write TestError_WithoutCause for Error() without cause
    - Write TestUnwrap for error unwrapping
    - Write TestIsNotFound for IsNotFound function
    - Write TestIsConflict for IsConflict function
    - Write TestIsUnauthorized for IsUnauthorized function
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9, 6.10, 6.11, 6.12, 6.13_
  
  - [ ]* 3.6 Write property tests for Typed Errors
    - **Property 21: Typed Error Type Field** - verify each constructor sets correct Type
    - **Property 22: Typed Error Format** - verify Error() string format
    - **Property 23: Typed Error Unwrap** - verify Unwrap returns Cause
    - **Property 24: Typed Error Type Checking** - verify Is* functions for each type
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9, 6.10, 6.11, 6.12, 6.13**

- [ ] 4. Checkpoint - Ensure domain and utility tests pass
  - Run `go test ./internal/domain/entity/... ./internal/pkg/...` to verify all tests pass
  - Verify coverage targets: Domain Entities 90%+, JWT Manager 85%+, Password Hasher 85%+, Typed Errors 90%+
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 5. Implement Application Service Tests
  - [ ] 5.1 Create Login service tests
    - Write TestLogin_Success for successful login with valid credentials
    - Write TestLogin_NonExistentEmail for non-existent email error
    - Write TestLogin_IncorrectPassword for wrong password error
    - Write TestLogin_SessionEviction for session eviction when limit reached
    - Create MockAccountProvider and MockSessionManager for login tests
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7_
  
  - [ ] 5.2 Create Refresh service tests
    - Write TestRefresh_CurrentToken for refresh with current token
    - Write TestRefresh_GracePeriod for refresh with prev token in grace period
    - Write TestRefresh_ReuseAttack for reuse attack detection
    - Write TestRefresh_InvalidTokenFormat for invalid token format
    - Write TestRefresh_RevokedSession for revoked session
    - Write TestRefresh_FingerprintMismatch for fingerprint mismatch
    - Write TestRefresh_SessionNotFound for session not found
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7, 8.8, 8.9_
  
  - [ ] 5.3 Create Logout service tests
    - Write TestLogout_AccessToken for logout with access token
    - Write TestLogout_RefreshToken for logout with refresh token
    - Write TestLogout_SessionID for logout with session ID
    - Write TestLogout_NoTokenOrSessionID for error when no identifier
    - Write TestLogoutAll for logout all sessions
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_
  
  - [ ] 5.4 Create Register service tests
    - Write TestRegister_Success for successful registration
    - Write TestRegister_DuplicateEmail for duplicate email error
    - Create MockAccountCreator for register tests
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_
  
  - [ ] 5.5 Create Introspect service tests
    - Write TestIntrospect_ValidToken for valid token introspection
    - Write TestIntrospect_ExpiredToken for expired token introspection
    - Write TestIntrospect_InvalidToken for invalid token introspection
    - Write TestIntrospect_BlacklistedToken for blacklisted token introspection
    - Write TestIntrospect_BlacklistError for blacklist check error
    - Create MockSessionChecker for introspect tests
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6_

- [ ] 6. Final checkpoint - Ensure all tests pass
  - Run `go test ./...` to verify all tests pass
  - Verify coverage targets: Application Services 80%+
  - Generate coverage report with `go test -cover ./...`
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from design document
- Unit tests validate specific examples and edge cases
- Test files are co-located with source files following Go convention
- Mock implementations use testify/mock for verification
- Table-driven tests are the primary pattern for multiple scenarios

## Coverage Targets

| Component | Target Coverage |
|-----------|-----------------|
| Domain Entities | 90%+ |
| JWT Manager | 85%+ |
| Password Hasher | 85%+ |
| Typed Errors | 90%+ |
| Application Services | 80%+ |
