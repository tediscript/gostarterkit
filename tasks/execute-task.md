# Task Execution Script for Go Starter Kit

## Purpose
This script executes tasks from the PRD systematically, marking completion in the PRD as work progresses.

## Execution Workflow

### Step 1: Read PRD
Read `tasks/prd-go-starter-kit.md` to understand all user stories and their acceptance criteria.

### Step 2: Identify Next Pending Task
Find the first incomplete user story (US-XXX) that has unchecked acceptance criteria.

**Priority Order:**
1. US-001 through US-015 in numerical order
2. Within each user story, complete acceptance criteria in the order listed

**Status Check:**
- `[ ]` = Pending (needs completion)
- `[x]` = Completed (already done)

### Step 3: Analyze Task Requirements
For the selected task:
1. Read the **Description** to understand the goal
2. Review all **Acceptance Criteria** with `[ ]` status
3. Identify any **dependencies** on previous tasks (which should be completed)
4. Note **environment variables** mentioned (use contextual naming per AGENTS.md)
5. Review **testing requirements** (unit, integration, negative, edge cases)

### Step 4: Implement the Task
**Implementation Guidelines:**
- Follow idiomatic Go conventions (see AGENTS.md)
- Use stdlib libraries unless external dependency is strongly justified
- Follow project structure: `cmd/app/`, `internal/`, `templates/`
- Use contextual environment variable names (e.g., `HTTP_PORT`, not `PORT`)
- Write tests for critical paths (unit + integration + negative + edge cases)
- Use structured logging with correlation IDs
- Handle errors gracefully

**Development Workflow:**
1. Create/edit necessary files in appropriate directories
2. Write implementation code
3. Write tests (unit and integration)
4. Run tests to verify: `go test ./...`
5. Run linter: `go vet ./...`
6. Build application: `make build`
7. Test functionality manually if applicable

### Step 5: Verify Completion
For each acceptance criterion in the task:

**Verification Methods:**
- **Code tests:** Run test suite and verify all pass
- **Integration tests:** Test endpoints/handlers with HTTP client
- **Manual verification:** Test via browser/curl as specified
- **Build verification:** Ensure `go build` succeeds

**Testing Categories:**
- **Unit tests:** Individual functions/methods
- **Integration tests:** Middleware, handlers, endpoints
- **Negative cases:** Invalid inputs, error conditions
- **Edge cases:** Boundary values, special characters, concurrent access

### Step 6: Mark Completion
Update `tasks/prd-go-starter-kit.md`:
1. Change `[ ]` to `[x]` for each completed acceptance criterion
2. Only mark criteria as complete when ALL verification passes
3. Do NOT mark criteria complete if tests fail or manual verification fails
4. Be conservative - if uncertain, mark as incomplete

### Step 7: Final Validation
Before considering the task complete:
- All acceptance criteria have `[x]` checked
- All tests pass: `go test ./... -v`
- Application builds successfully: `make build`
- Application runs without errors: `make run` (or `make watch` for dev mode)
- Manual verification steps pass (if applicable)

### Step 8: Report Completion
Provide summary of work done:
- User story ID and title
- Files created/modified
- Tests written and passing
- Verification steps performed
- Any issues encountered and resolved

## Example Execution

**Input:** Next pending task is US-003: HTML template rendering

**Process:**
1. Read US-003 requirements from PRD
2. Identify acceptance criteria: create templates/, implement parsing, caching, hot-reload, etc.
3. Create `templates/base.html` and `templates/home.html`
4. Implement template parser in `internal/handlers/handlers.go`
5. Add template caching logic (production) and hot-reload (development)
6. Write unit tests for template parsing
7. Write integration tests for caching/hot-reload
8. Run `go test ./...` - all pass
9. Run `make build` - succeeds
10. Run `make run` - server starts
11. Test via browser: `http://localhost:8880/` - renders correctly
12. Update PRD: mark all US-003 criteria as `[x]`
13. Commit the work using use_skill commit method
14. Push the work to current branch

**Output:** US-003 complete, all tests passing, application verified

## Important Notes

- **Do not skip tasks** - execute in numerical order (US-003 → US-004 → ...)
- **Do not skip acceptance criteria** - complete all listed criteria before marking complete
- **Dependencies matter** - if a task depends on previous work, ensure that work is complete
- **Quality over speed** - prefer complete, tested implementation over partial, untested code
- **Context matters** - always reference AGENTS.md for conventions and guidelines
- **Environment variables** - always use contextual naming (e.g., `HTTP_PORT` not `PORT`)

## Current Status

When executing this script, start from the first unchecked acceptance criterion in the first incomplete user story.

**Last completed:** US-002 (configuration with _FILE support)
**Next task:** US-003 (HTML template rendering)

---
*This script is designed for systematic, reliable execution of the Go Starter Kit PRD tasks.*