# FullDash Architecture

## Overview
FullDash is a lightweight project dashboard for tracking freelance work, revenue splits, and payments.

## Tech Stack
- **Backend:** Go 1.25+ with Chi router
- **Templating:** Templ (type-safe HTML)
- **Frontend:** HTMX 2.0 + vanilla CSS
- **Database:** SQLite (modernc.org/sqlite - pure Go)
- **Payments:** Stripe webhooks

## Project Structure

```
cmd/fullstacked/
  main.go              # Entry point, routes, middleware

internal/
  handlers/
    web.go             # HTTP handlers (dashboard, CRUD)
    forms.go           # Form parsing helpers (DRY)
    stripe.go          # Stripe webhook handlers
  
  models/
    project.go         # Domain models (Project, Contribution, etc.)
  
  store/
    interface.go       # Store interface (for mocking)
    queries.go         # SQL query constants (DRY)
    db.go              # Core DB operations
    contributions.go   # Contribution operations
    metrics.go         # Business logic for metrics
  
  templates/
    *.templ            # Templ templates (compile to *_templ.go)

static/css/
  main.css             # Vanilla CSS, dark mode

data/
  fulldash.db          # SQLite database
```

## Data Flow

```
HTTP Request
    ↓
Chi Router
    ↓
Handler (internal/handlers/)
    - Parse form/query params
    - Call Store methods
    - Render template or return error
    ↓
Store (internal/store/)
    - Execute SQL queries
    - Return models
    ↓
Template (internal/templates/)
    - Generate HTML
    ↓
HTTP Response
```

## Key Design Decisions

### 1. Store Interface
- Defined in `store/interface.go`
- Enables unit testing with mocks
- Compile-time verification: `var _ Store = (*DB)(nil)`

### 2. DRY SQL Queries
- All SQL in `store/queries.go` as constants
- Column lists defined once, reused everywhere
- Generic `scanAll()` helper for row scanning

### 3. Form Parsing
- Centralized in `handlers/forms.go`
- `ParsedForm` struct holds all values
- Helper methods: `toProject()`, `applyTo()`, `saveContributions()`

### 4. Revenue Split Logic
```
If both Noor AND Ahmad have hours logged:
    Split = hours_ratio(project.revenue)
Else:
    Split = ownership_rule(project.secured_by)
```

### 5. HTMX Patterns
- Full page render on initial load
- Partial swaps for HTMX requests (`HX-Request` header check)
- Modal forms with `hx-target="#modal"`

## Database Schema

```sql
projects:
  - id (PK)
  - client (text, required)
  - description (text)
  - revenue (real, default 0)
  - status (new|in_progress|done|paid)
  - secured_by (noor|ahmad|both)
  - stripe_payment_id (text, optional)
  - created_at (datetime)

contributions:
  - id (PK)
  - project_id (FK → projects)
  - owner (noor|ahmad)
  - hours (real)
  - notes (text)
  - UNIQUE(project_id, owner)
```

## Environment Variables

```bash
PORT=8080                    # Server port
DB_PATH=data/fulldash.db     # Database file path
STRIPE_SECRET_KEY=           # For future Stripe API calls
STRIPE_WEBHOOK_SECRET=       # For webhook verification
```

## Testing Strategy

### Unit Tests (Todo)
- Mock `Store` interface
- Test handler logic without real DB
- Test revenue split calculations

### Integration Tests (Todo)
- SQLite in-memory database
- Full request/response cycle

### Manual Testing
```bash
# Start server
./fullstacked

# Health check
curl http://localhost:8080/health

# Screenshot verification
puppeteer screenshot http://localhost:8080 /tmp/test.png
```

## Common Tasks

### Adding a New Field to Projects
1. Update `models/project.go`
2. Update schema in `store/db.go` (migrate func)
3. Update `store/queries.go` (columns constant)
4. Update form parsing in `handlers/forms.go`
5. Update templates in `internal/templates/`
6. Run `templ generate`

### Adding a New Handler
1. Add route in `cmd/fullstacked/main.go`
2. Implement handler in `handlers/*.go`
3. Add template if needed in `templates/*.templ`
4. Run `templ generate`

## LLM Context Hints

When modifying this codebase:
1. **Check interfaces first** - Store interface defines all DB operations
2. **SQL changes go in queries.go** - Don't inline SQL
3. **Form handling use helpers** - Use `parseProjectForm()` and methods
4. **Templates need regeneration** - Run `templ generate` after changes
5. **Test with screenshots** - Use puppeteer for visual verification
