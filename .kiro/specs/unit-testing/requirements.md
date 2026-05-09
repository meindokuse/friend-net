# Requirements Document

## Introduction

This spec defines requirements for implementing comprehensive unit testing in the auth-service Go application. The auth-service implements JWT authentication, refresh token rotation, OAuth (Google), and Outbox pattern using Clean Architecture. The MVP is complete but lacks tests. This spec covers domain entity tests, utility tests, and application service tests using the standard `testing` package with `testify` assertions and mocks.

## Glossary

- **Test_Suite**: A collection of related test cases organized using `testify/suite`
- **Mock**: A test double that implements an interface for verifying interactions, created using `testify/mock`
- **Domain_Entity**: A core business entity in the Clean Architecture domain layer (Account, Session, RefreshPair)
- **Application_Service**: A use case implementation in the Clean Architecture application layer
- **JWT_Manager**: The jwt.Manager component responsible for access and refresh token operations
- **Password_Hasher**: The pass.Hasher component responsible for bcrypt password hashing
- **Typed_Error**: A structured error from the terror package with type, message, and cause
- **RefreshPair**: A pair of refresh token hashes (current and prev) for rotation and grace period handling
- **Grace_Period**: A time window during which the previous refresh token remains valid for retry scenarios
- **Reuse_Attack**: A security threat where an attacker attempts to use a previously used refresh token
- **Session_Eviction**: The process of removing the oldest session when a user exceeds the maximum session limit

## Requirements

### Requirement 1: Account Entity Tests

**User Story:** As a developer, I want comprehensive tests for the Account entity, so that I can verify account creation, validation, and state changes work correctly.

#### Acceptance Criteria

1. WHEN NewAccount is called with valid email and password hash, THE Account_Entity SHALL return an Account with a valid UUID, IsActive set to true, and timestamps initialized
2. WHEN NewAccount is called with any parameters, THE Account_Entity SHALL set CreatedAt and UpdatedAt to approximately the same UTC timestamp
3. WHEN UpdateLastLogin is called on an Account, THE Account_Entity SHALL set LastLoginAt to the current UTC timestamp and update UpdatedAt
4. FOR ALL Account instances, THE Account_Entity SHALL have LastLoginAt as nil until UpdateLastLogin is called

### Requirement 2: Session Entity Tests

**User Story:** As a developer, I want comprehensive tests for the Session entity, so that I can verify session lifecycle management and state transitions.

#### Acceptance Criteria

1. WHEN NewSession is called with valid parameters, THE Session_Entity SHALL return a Session with Status set to "active", CreatedAt and LastSeenAt set to approximately the same timestamp, and ExpiresAt calculated from TTL
2. WHEN IsActive is called on a Session with Status "active" and ExpiresAt in the future, THE Session_Entity SHALL return true
3. WHEN IsActive is called on a Session with Status "revoked", THE Session_Entity SHALL return false
4. WHEN IsActive is called on a Session with ExpiresAt in the past, THE Session_Entity SHALL return false
5. WHEN Revoke is called on a Session, THE Session_Entity SHALL set Status to "revoked"
6. WHEN UpdateLastSeen is called on a Session, THE Session_Entity SHALL update LastSeenAt to the current UTC timestamp

### Requirement 3: RefreshPair Entity Tests

**User Story:** As a developer, I want comprehensive tests for the RefreshPair entity, so that I can verify refresh token matching, rotation, and grace period logic.

#### Acceptance Criteria

1. WHEN Match is called with a hash equal to Current, THE RefreshPair_Entity SHALL return RefreshMatchCurrent
2. WHEN Match is called with a hash equal to Prev and current time is before PrevExpiresAt, THE RefreshPair_ENTITY SHALL return RefreshMatchPrev
3. WHEN Match is called with a hash equal to Prev and current time is after PrevExpiresAt, THE RefreshPair_Entity SHALL return RefreshMatchNone
4. WHEN Match is called with a hash matching neither Current nor Prev, THE RefreshPair_Entity SHALL return RefreshMatchNone
5. WHEN Rotate is called with a new hash and grace period, THE RefreshPair_Entity SHALL move Current to Prev, set PrevExpiresAt to now + grace period, and set Current to the new hash
6. WHEN SetCurrent is called with a new hash, THE RefreshPair_Entity SHALL update Current without modifying Prev or PrevExpiresAt

### Requirement 4: JWT Manager Tests

**User Story:** As a developer, I want comprehensive tests for the JWT Manager, so that I can verify token generation, verification, and security properties.

#### Acceptance Criteria

1. WHEN NewManager is called with secret key shorter than 32 characters, THE JWT_Manager SHALL return an error
2. WHEN NewManager is called with refresh secret shorter than 32 characters, THE JWT_Manager SHALL return an error
3. WHEN NewManager is called with valid secrets of 32 or more characters, THE JWT_Manager SHALL return a Manager instance
4. WHEN GenerateAccessToken is called with valid sessionID and userID, THE JWT_Manager SHALL return a signed JWT string
5. WHEN VerifyAccessToken is called with a valid token, THE JWT_Manager SHALL return AccessClaims with correct SessionID and Subject
6. WHEN VerifyAccessToken is called with an expired token, THE JWT_Manager SHALL return ErrTokenExpired
7. WHEN VerifyAccessToken is called with an invalid signature, THE JWT_Manager SHALL return an error containing ErrInvalidToken
8. WHEN GenerateRefreshToken is called with a sessionID, THE JWT_Manager SHALL return a token in format "sessionID.randomPart" where randomPart is at least 32 characters
9. WHEN ParseRefreshToken is called with a valid token, THE JWT_Manager SHALL return the sessionID and randomPart separately
10. WHEN ParseRefreshToken is called with a token missing the separator, THE JWT_Manager SHALL return ErrInvalidToken
11. WHEN HashRefreshToken is called with the same randomPart multiple times, THE JWT_Manager SHALL return the same hash (deterministic)
12. WHEN HashFingerprint is called with the same fingerprint multiple times, THE JWT_Manager SHALL return the same hash (deterministic)
13. WHEN HashRefreshToken is called with different randomParts, THE JWT_Manager SHALL return different hashes

### Requirement 5: Password Hasher Tests

**User Story:** As a developer, I want comprehensive tests for the Password Hasher, so that I can verify password hashing and comparison operations.

#### Acceptance Criteria

1. WHEN Hash is called with any password, THE Password_Hasher SHALL return a bcrypt hash string
2. WHEN Compare is called with a hash and its original password, THE Password_Hasher SHALL return nil
3. WHEN Compare is called with a hash and a different password, THE Password_Hasher SHALL return an error
4. WHEN Match is called with a hash and its original password, THE Password_Hasher SHALL return true
5. WHEN Match is called with a hash and a different password, THE Password_Hasher SHALL return false
6. WHEN New is called with cost below bcrypt.MinCost, THE Password_Hasher SHALL use bcrypt.DefaultCost
7. WHEN New is called with cost above bcrypt.MaxCost, THE Password_Hasher SHALL use bcrypt.DefaultCost

### Requirement 6: Typed Error Tests

**User Story:** As a developer, I want comprehensive tests for the terror package, so that I can verify error creation, type checking, and error chain behavior.

#### Acceptance Criteria

1. WHEN NewNotFoundErr is called with message and cause, THE Typed_Error SHALL return an Error with Type "NOT_FOUND", the specified Message, and Cause set
2. WHEN NewConflictErr is called with message and cause, THE Typed_Error SHALL return an Error with Type "CONFLICT"
3. WHEN NewUnauthorizedErr is called with message and cause, THE Typed_Error SHALL return an Error with Type "UNAUTHORIZED"
4. WHEN NewBadRequestErr is called with message and cause, THE Typed_Error SHALL return an Error with Type "BAD_REQUEST"
5. WHEN NewInternalErr is called with message and cause, THE Typed_Error SHALL return an Error with Type "INTERNAL"
6. WHEN NewForbiddenErr is called with message and cause, THE Typed_Error SHALL return an Error with Type "FORBIDDEN"
7. WHEN Error method is called on an Error with Cause, THE Typed_Error SHALL return a string in format "TYPE: message: cause"
8. WHEN Error method is called on an Error without Cause, THE Typed_Error SHALL return a string in format "TYPE: message"
9. WHEN Unwrap is called on an Error, THE Typed_Error SHALL return the Cause
10. WHEN IsNotFound is called with a NOT_FOUND error, THE Typed_Error SHALL return true
11. WHEN IsNotFound is called with any other error type, THE Typed_Error SHALL return false
12. WHEN IsConflict is called with a CONFLICT error, THE Typed_Error SHALL return true
13. WHEN IsUnauthorized is called with an UNAUTHORIZED error, THE Typed_Error SHALL return true

### Requirement 7: Login Service Tests

**User Story:** As a developer, I want comprehensive tests for the Login Service, so that I can verify authentication logic and session management.

#### Acceptance Criteria

1. WHEN Login is called with valid credentials, THE Login_Service SHALL return Result with AccessToken, RefreshToken, and AccountID
2. WHEN Login is called with non-existent email, THE Login_Service SHALL return an UNAUTHORIZED error with message "invalid credentials"
3. WHEN Login is called with incorrect password, THE Login_Service SHALL return an UNAUTHORIZED error with message "invalid credentials"
4. WHEN Login is called and active session count equals maxSessionsPerUser (3), THE Login_Service SHALL revoke the oldest session before creating a new one
5. WHEN Login is called with valid credentials, THE Login_Service SHALL call SessionManager.Create with a valid Session entity
6. WHEN Login is called with valid credentials, THE Login_Service SHALL call SessionManager.SaveRefreshPair with a RefreshPair containing the hashed refresh token
7. WHEN Login is called with valid credentials, THE Login_Service SHALL set FingerprintHash on the session using JWT Manager's HashFingerprint

### Requirement 8: Refresh Service Tests

**User Story:** As a developer, I want comprehensive tests for the Refresh Service, so that I can verify token rotation, grace period handling, and security measures.

#### Acceptance Criteria

1. WHEN Refresh is called with valid current refresh token, THE Refresh_Service SHALL return new AccessToken and RefreshToken
2. WHEN Refresh is called with valid current refresh token, THE Refresh_Service SHALL call RefreshPair.Rotate to update the pair
3. WHEN Refresh is called with prev token within grace period, THE Refresh_Service SHALL return new tokens without overwriting Prev
4. WHEN Refresh is called with prev token within grace period, THE Refresh_Service SHALL call RefreshPair.SetCurrent only
5. WHEN Refresh is called with unknown token (no match), THE Refresh_Service SHALL revoke the session and return UNAUTHORIZED error with message "token reuse detected"
6. WHEN Refresh is called with invalid token format, THE Refresh_Service SHALL return UNAUTHORIZED error with message "invalid refresh token"
7. WHEN Refresh is called for a revoked session, THE Refresh_Service SHALL return UNAUTHORIZED error with message "session revoked"
8. WHEN Refresh is called with mismatched fingerprint, THE Refresh_Service SHALL return UNAUTHORIZED error with message "fingerprint mismatch"
9. WHEN Refresh is called for a session not found in storage, THE Refresh_Service SHALL return NOT_FOUND error with message "session not found"

### Requirement 9: Logout Service Tests

**User Story:** As a developer, I want comprehensive tests for the Logout Service, so that I can verify session termination and token blacklisting.

#### Acceptance Criteria

1. WHEN Logout is called with AccessToken, THE Logout_Service SHALL extract sessionID from the token and revoke the session
2. WHEN Logout is called with RefreshToken, THE Logout_Service SHALL extract sessionID from the token and revoke the session
3. WHEN Logout is called with SessionID directly, THE Logout_Service SHALL revoke the session with that ID
4. WHEN Logout is called with AccessToken, THE Logout_Service SHALL blacklist the access token's jti with remaining TTL
5. WHEN Logout is called without any token or sessionID, THE Logout_Service SHALL return BAD_REQUEST error with message "session id required"
6. WHEN LogoutAll is called with accountID, THE Logout_Service SHALL call SessionManager.RevokeAllByAccountID with that accountID

### Requirement 10: Register Service Tests

**User Story:** As a developer, I want comprehensive tests for the Register Service, so that I can verify account creation and duplicate handling.

#### Acceptance Criteria

1. WHEN Register is called with new email and valid password, THE Register_Service SHALL return the new account's ID
2. WHEN Register is called with new email and valid password, THE Register_Service SHALL call AccountCreator.CreateWithOutbox with Account and OutboxEvent
3. WHEN Register is called with an email that already exists, THE Register_Service SHALL return CONFLICT error with message "account already exists"
4. WHEN Register is called with valid data, THE Register_Service SHALL create an OutboxEvent with type "account.created"
5. WHEN Register is called with valid data, THE Register_Service SHALL hash the password using Hasher.Hash

### Requirement 11: Introspect Service Tests

**User Story:** As a developer, I want comprehensive tests for the Introspect Service, so that I can verify token validation and blacklist checking.

#### Acceptance Criteria

1. WHEN Introspect is called with a valid access token, THE Introspect_Service SHALL return Result with Active true, AccountID, and SessionID
2. WHEN Introspect is called with an expired token, THE Introspect_Service SHALL return Result with Active false
3. WHEN Introspect is called with an invalid token, THE Introspect_Service SHALL return Result with Active false
4. WHEN Introspect is called with a blacklisted token, THE Introspect_Service SHALL return Result with Active false
5. WHEN Introspect is called with a valid token, THE Introspect_Service SHALL call SessionChecker.IsBlacklisted with the token's jti
6. WHEN SessionChecker.IsBlacklisted returns an error, THE Introspect_Service SHALL return INTERNAL error with message "check blacklist"

### Requirement 12: Test File Organization

**User Story:** As a developer, I want tests placed alongside source files, so that I can easily find and run tests for specific components.

#### Acceptance Criteria

1. THE Test_File for domain entities SHALL be located in `internal/domain/entity/*_test.go`
2. THE Test_File for JWT Manager SHALL be located in `internal/pkg/jwt/manager_test.go`
3. THE Test_File for Password Hasher SHALL be located in `internal/pkg/pass/pass_test.go`
4. THE Test_File for terror package SHALL be located in `internal/pkg/terror/errors_test.go`
5. THE Test_File for Login Service SHALL be located in `internal/application/service/auth/login/service_test.go`
6. THE Test_File for Refresh Service SHALL be located in `internal/application/service/auth/refresh/service_test.go`
7. THE Test_File for Logout Service SHALL be located in `internal/application/service/auth/logout/service_test.go`
8. THE Test_File for Register Service SHALL be located in `internal/application/service/auth/register/service_test.go`
9. THE Test_File for Introspect Service SHALL be located in `internal/application/service/auth/introspect/service_test.go`

### Requirement 13: Test Technology Stack

**User Story:** As a developer, I want to use standard Go testing tools with testify extensions, so that tests are idiomatic and maintainable.

#### Acceptance Criteria

1. THE Test_Code SHALL use the standard `testing` package for test function signatures
2. THE Test_Code SHALL use `testify/assert` for assertions
3. THE Test_Code SHALL use `testify/mock` for implementing mock interfaces
4. THE Test_Code MAY use `testify/suite` for organizing related test cases into suites
5. THE Mock_Implementation SHALL implement the exact interface methods defined in each service's dependencies
