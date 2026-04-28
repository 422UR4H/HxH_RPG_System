# Domain Use Case Tests — Design Spec

## Goal

Provide comprehensive unit test coverage for all 16 domain use cases across 6 packages: `scenario`, `campaign`, `match`, `auth`, `submission`, and `enrollment`.

## Architecture

### Mock Infrastructure

All tests use manual mock implementations located in `internal/domain/testutil/`. Each mock struct has configurable function fields (e.g., `CreateScenarioFn`) that allow tests to control return values and errors per test case.

Mock files:
- `mock_scenario_repo.go` — implements `scenario.IRepository`
- `mock_campaign_repo.go` — implements `campaign.IRepository`
- `mock_match_repo.go` — implements `match.IRepository`
- `mock_auth_repo.go` — implements `auth.IRepository`
- `mock_session_repo.go` — implements `session.IRepository`
- `mock_submission_repo.go` — implements `submission.IRepository`
- `mock_enrollment_repo.go` — implements `enrollment.IRepository`
- `mock_character_sheet_repo.go` — implements `charactersheet.IRepository`

### Test Strategy

- **External test packages** (`package scenario_test`) for black-box testing
- **Table-driven tests** with `t.Run()` sub-tests
- **Error comparison** via `.Error()` string matching (because domain errors wrap via `domain.NewValidationError`)
- **Gateway errors** imported directly to trigger use case error translation paths (e.g., `scenarioPg.ErrScenarioNotFound` → `scenario.ErrScenarioNotFound`)

## Use Cases Covered

### Scenario (3 UCs, 14 test cases)
| Use Case | Test Cases |
|----------|-----------|
| CreateScenario | success, name too short, name too long, brief desc too long, name exists, repo errors |
| GetScenario | success as owner, not found, insufficient permissions, repo error |
| ListScenarios | success with results, empty, repo error |

### Campaign (3 UCs, 13 test cases)
| Use Case | Test Cases |
|----------|-----------|
| CreateCampaign | success (with/without scenario), name length, start date, brief desc, max limit, scenario not found, repo error |
| GetCampaign | success owner, success public other user, private denied, not found, repo error |
| ListCampaigns | success with results, empty, repo error |

### Match (4 UCs, 14 test cases)
| Use Case | Test Cases |
|----------|-----------|
| CreateMatch | success, title length, brief desc, game start past/future, campaign not found, not owner, story start bounds |
| GetMatch | success owner, success public, private denied, not found, repo error |
| ListMatches | success, empty, repo error |
| ListPublicUpcoming | success, empty, repo error |

### Auth (2 UCs, 23 test cases)
| Use Case | Test Cases |
|----------|-----------|
| Register | success, missing nick, nick length, missing email, email length, missing password, missing confirm, password length, mismatch, nick exists, email exists, repo errors |
| Login | success (bcrypt verification), missing email, email length, missing password, password length, email not found, wrong password |

### Submission (3 UCs, 14 test cases)
| Use Case | Test Cases |
|----------|-----------|
| Submit | success, sheet not found, not owner, already submitted, campaign not found, master self-submit, repo error |
| Accept | success, submission not found, campaign not found, not master |
| Reject | success, submission not found, not master |

### Enrollment (1 UC, 9 test cases)
| Use Case | Test Cases |
|----------|-----------|
| Enroll | success, sheet not found, not owner, nil player UUID, already enrolled, match not found, not in campaign, nil campaign UUID, repo error |

## Design Decisions

1. **No external mock frameworks** — manual mocks with function fields are simpler, more transparent, and produce zero dependencies.

2. **Error string comparison** — since domain errors use `domain.NewValidationError(errors.New(...))`, the resulting errors implement `error` interface. We compare via `.Error()` string method rather than pointer equality.

3. **Gateway error imports in tests** — tests import gateway-layer errors (e.g., `pgUser.ErrEmailNotFound`) to trigger error translation paths in use cases. This is intentional and verifies the UC correctly translates infrastructure errors to domain errors.

4. **Login test uses real bcrypt** — the Login UC calls `bcrypt.CompareHashAndPassword` directly, so tests generate real bcrypt hashes. This ensures the actual crypto path is exercised.

## Files Created

```
internal/domain/testutil/
├── doc.go
├── mock_auth_repo.go
├── mock_campaign_repo.go
├── mock_character_sheet_repo.go
├── mock_enrollment_repo.go
├── mock_match_repo.go
├── mock_scenario_repo.go
├── mock_session_repo.go
└── mock_submission_repo.go

internal/domain/scenario/scenario_test.go
internal/domain/campaign/campaign_test.go
internal/domain/match/match_uc_test.go
internal/domain/auth/auth_test.go
internal/domain/submission/submission_test.go
internal/domain/enrollment/enrollment_test.go
```

## Test Results

- **21 packages pass** (all new + all pre-existing entity tests)
- **1 pre-existing failure** (`turn/engine_test.go` — known broken from Turn/Round refactoring, excluded from scope)
- **87 total new test cases** across 16 use cases
