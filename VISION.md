# hput Vision & Context Document

**Date:** 2026-01-03
**Purpose:** Align on project vision with proper persistent data architecture

---

## Current State Analysis

### What We Have
- **Core Concept:** HTTP server programmable via HTTP (PUT code/content, GET/POST to execute)
- **Key Features:**
  - PUT static files (HTML, images, etc.) and retrieve them
  - PUT JavaScript to paths and execute on any HTTP verb
  - Express-style `request`/`response` objects in JavaScript runtime
  - `/dump` endpoint to export configuration
  - Storage backends: memory, local disk (bbolt), S3
  - Locked mode to prevent modifications after deployment

### Technology Stack (v0.1 - OUTDATED)
- **Go Version:** 1.21 (released August 2023)
- **Current Go:** 1.25.5 (as of Jan 2026)
- **JavaScript Runtime:** v8go (V8 JavaScript engine bindings - need to check for latest version)
- **Database:** bbolt (embedded key-value store)
- **Cloud:** AWS SDK v2 for S3

### Planned Upgrades (Future Tasks)
- [ ] Install Go 1.25.5 on M4 MacBook
- [ ] Upgrade project to Go 1.25.5
- [ ] Apply Go 1.25.5 syntax improvements to codebase
- [ ] Upgrade v8go to latest version
- [ ] Review and apply key syntax improvements from v8go updates

### Git Status Issues
```
Deleted from index:
- .gitignore, docker-compose.yml, dockerfile
- All source files: hput.go, main.go, all package files
- Service layer: discsaver, httpserver, javascript, logger, mapsaver, s3saver, service

Untracked:
- .vscode/, docs/, v0.1/
```

**Analysis:** Looks like work-in-progress was moved to `v0.1/` but changes not committed properly. Current branch state is broken.

---

## Problems Identified

### 1. Persistent Data Layer - Not Properly Architected
**Current Issues:**
- bbolt is key-value only, no indexing, no queries
- No schema/data model design
- Can't do "list all paths" efficiently (needed for `/dump`, path enumeration)
- JavaScript can't query across paths
- No metadata storage (timestamps, content-type, size, etc.)

**Example from README line 19:**
> `// TODO: add ?list=paths to the API`

This is hard with current architecture.

### 2. Bugs (Need Investigation)
- No specifics mentioned, need to identify
- Testing appears incomplete (tests exist but coverage unknown)

### 3. Branch Divergence
- Deleted files suggest major refactor in progress
- Unclear what changes were intended
- v0.1 appears to be the actual code

### 4. Go Version Outdated
- Go 1.21 is now 2+ years old
- Modern Go (1.23+) has improved:
  - Range over functions (iter package)
  - Better error handling patterns
  - Performance improvements
  - Standard library enhancements

---

## Architecture Decisions (RESOLVED)

### 1. Data Model ✓
**Per-path metadata:**
- Timestamps (created, updated)
- Content-type (though JavaScript paths don't have a "returned" content-type, they execute)
- **Dynamic/flexible schema** - ability to add more keys as service evolves

**Not supported:**
- ❌ Namespaces (assume user already authenticated to service)
- ❌ Versioning (too complex)

**Path hierarchy:**
- ✅ Support paths and sub-paths
- ⚠️ Path-based access control does NOT exist in v0.1 (only localhost filtering)
- 🔜 Need to design path-based access control for v0.2

**HTTP verbs:**
- ✅ All verbs (POST, GET, DELETE, etc.) pass into JavaScript like AWS Lambda (VERIFIED ✓)
- ✅ PUT is reserved for uploading content/code (VERIFIED ✓)

### 2. Storage Strategy ✓
**Key requirements:**
- Key/value first with flexible schema
- NOT optimized for hyperscale - designed for 1 pod operation
- `/dump` can take time, not a performance concern
- Use Go interfaces to allow developers to add storage backends

**Storage model options to evaluate:**
- bbolt (current) - not ruled out
- Alternatives TBD - need to research key/value stores with flexible schema
- SQLite - uncertain, evaluate pros/cons

**Architecture pattern:**
- Storage backend abstraction via Go interfaces
- Pluggable storage implementations

### 3. JavaScript Capabilities ✓
**Shared data access:**
- ✅ JS needs to read shared storage (KEY ARCHITECTURAL CHALLENGE - see below)
- ❌ JS should NOT read/write other paths
- ✅ Transactions are atomic in service already (verify)
- ❌ Don't need ACID refinements beyond current atomicity

### 4. Security Model ✓
**Access control:**
- ⚠️ v0.1 only has localhost vs non-local filtering
- 🔜 Path-based access control needs to be designed for v0.2
- ❌ No authn/authz exists in v0.1
- ✅ v8go JavaScript sandboxing (VERIFIED ✓ - already exists, keep it)
- ❌ Do NOT validate content-type

### 5. Primary Use Cases ✓
**Priority:**
1. **Personal rapid prototyping** - Quickest way to get people to try it
2. **Glitch alternative (long-term)** - Open source advantage over freemium services

**Not priorities right now:**
- Mock/test server (nice to have)
- Education platform (nice to have)
- Production micro-services (maybe later)

### 6. MVP vs Future
**Status:** TBD - need to define after resolving shared storage architecture

---

## Code Review Findings (v0.1)

### ✅ Verified Features
1. **HTTP verb handling** - Confirmed in [httpserver.go](v0.1/httpserver/httpserver.go:62-76)
   - PUT uploads content/code (unless in locked mode)
   - All other verbs (GET, POST, DELETE, etc.) execute via `Run()`

2. **Express-style request/response** - Confirmed in [http.go](v0.1/javascript/http.go)
   - `request` object: get, cookies, hostname, ip, method, path, protocol, query, body, headers
   - `response` object: append, cookie, json, location, redirect, send, sendStatus, set, status
   - Also includes `console.log` for logging
   - Also includes `fetch()` and `timers` via v8go-polyfills

3. **Storage implementation** - [discsaver/db.go](v0.1/discsaver/db.go)
   - Single bbolt bucket named "hput"
   - Key: path (string)
   - Value: JSON-marshaled `Runnable` struct (Type, Text, Binary)
   - `/dump` works via prefix scan using bbolt cursor (lines 145-169)

4. **Interface abstraction** - Confirmed in [service/service.go](v0.1/service/service.go:17-24)
   - `Saver` interface allows pluggable storage backends
   - `Interpreter` interface allows pluggable runtimes

### ❌ Features NOT Found (Needs Correction)
1. **Path-based access control** - Does NOT exist
   - Only has localhost vs non-local traffic filtering
   - No sub-path access control found
   - No authentication/authorization

2. **Shared storage for JavaScript** - Does NOT exist
   - JavaScript can only use `request`/`response` objects
   - Cannot read/write to storage
   - Cannot access other paths or shared data

### Current Data Model
```go
// From hput.go
type Runnable struct {
    Path   string // exact location of resource on this server
    Type   Input  // "Text", "Javascript", or "Binary"
    Text   string // for Text and Javascript types
    Binary []byte // for Binary type
}
```

**Limitations:**
- No timestamps
- No content-type metadata
- No flexible schema
- Only stores the content itself

---

## 🚨 KEY ARCHITECTURAL CHALLENGE: Shared Storage for JavaScript

### The Problem
JavaScript executing at one path needs to read/write shared data, but:
- Should NOT read/write other paths (security)
- Architecture must stay simple (like a Lego brick)
- Must work conceptually like a data store API

### The Question
**How do we give JavaScript functions access to a shared data store that is:**
1. **Simple** - Easy to understand and use
2. **Secure** - Can't access other paths' code
3. **Persistent** - Survives restarts
4. **Flexible** - Schema-less or dynamic schema
5. **Lego-like** - Adds value without complexity

### Proposed Solution: Subpath-Based Access Control

**Core Insight:** Use the path hierarchy for natural security boundaries.

**Access Model:**
- JavaScript at path `/X` can access ONLY `/X/*` (descendant paths)
- Relative paths in code (e.g., `'123'`) automatically resolve to `/X/123`
- Absolute paths work if within scope (e.g., `/X/123` from code at `/X`)
- Code CANNOT access paths outside its subtree

**Security Rules:**
1. **Scope check:** Code at `/users` can only access `/users/*`
2. **Type check:** Code cannot read other JavaScript (prevents code injection)
3. **Hierarchy:** Code at `/users/admin` can access `/users/admin/*` but NOT `/users/regular/*`

**API Design:**

#### Primary API: `hput` global object with relative paths
```javascript
// JavaScript at /users
const user = await hput.get('123')              // Reads /users/123
await hput.put('456', {name: 'Bob'})            // Writes /users/456
const all = await hput.list('*')                // Lists /users/*
await hput.delete('789')                        // Deletes /users/789

// Can also use absolute paths (if within scope)
const user = await hput.get('/users/123')       // Same as get('123')
```

#### Alternative: Leverage existing `fetch()` polyfill
```javascript
// Could also intercept fetch calls to localhost paths
const response = await fetch('123')             // Reads /users/123
const user = await response.json()

await fetch('456', {
  method: 'PUT',
  body: JSON.stringify({name: 'Bob'})
})
```

**Recommendation:** Implement `hput` global object first (simpler API), then optionally add `fetch()` interception for developers who prefer HTTP semantics.

### Requirements Resolved ✓
- ✅ Data storage is scoped to subtree (hierarchical security)
- ✅ NO forced path conventions - users choose their structure
- ✅ Relative paths make code simple and portable
- ✅ Directly relates to PUT/GET mechanics - same paths
- ✅ Can add complex permissions later (cross-path access, `/data/*`, etc.)

---

## Technical Decisions Needed

### Database Choice (Key/Value + Flexible Schema)
| Option | Pros | Cons |
|--------|------|------|
| **bbolt** (current) | Embedded, simple, fast, pure Go, buckets for organization | No secondary indexes, manual iteration for queries |
| **BadgerDB** | Embedded, fast, LSM tree, built-in MVCC | More complex than bbolt, larger dependency |
| **Pebble** | CockroachDB's KV store, fast, RocksDB-like | Heavier dependency, might be overkill |
| **SQLite** | Embedded, JSON support, SQL for queries | Schema overhead, less "key/value first" |
| **LevelDB** | Google's KV store, proven | No longer actively maintained |

### Code Organization
```
Option A: Keep v0.1 structure
cmd/hput/main.go
pkg/
  storage/    (interface + impls)
  runtime/    (JS execution)
  server/     (HTTP handlers)

Option B: Simpler monolith
cmd/hput/main.go
internal/
  storage.go
  javascript.go
  handlers.go

Option C: Domain-driven
cmd/hput/main.go
internal/
  paths/      (path management domain)
  runtime/    (JS runtime domain)
  api/        (HTTP API domain)
```

### Go Upgrade Path
1. Update to Go 1.24 or 1.23?
2. Update all dependencies (AWS SDK, v8go, bbolt, etc.)
3. Any new stdlib we can use?

---

## Persistent Data Design Proposals

### Option 1: Enhanced Key-Value (bbolt)
```
Buckets:
- content/{path} -> raw content bytes
- metadata/{path} -> JSON {timestamp, contentType, size, etag}
- index/by_timestamp -> sorted set of paths
- code/{path} -> JavaScript source
```

### Option 2: SQLite Schema
```sql
CREATE TABLE paths (
  id INTEGER PRIMARY KEY,
  path TEXT UNIQUE NOT NULL,
  content BLOB NOT NULL,
  content_type TEXT,
  is_code BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  size INTEGER,
  etag TEXT
);

CREATE INDEX idx_paths_updated ON paths(updated_at DESC);
CREATE INDEX idx_paths_type ON paths(content_type);
CREATE INDEX idx_paths_code ON paths(is_code);
```

### Option 3: Hybrid Approach
- SQLite for metadata and queryable index
- Filesystem or S3 for large blobs (images, videos)
- bbolt for hot cache

---

## Proposed Vision Statement

> **hput** is a programmable HTTP server that enables rapid web development through HTTP itself. Store content and code at any path via PUT, then access via GET or execute via any HTTP verb. Built for prototyping, learning, and lightweight deployments where simplicity trumps ceremony.

### Core Principles
1. **Zero Config Start:** `hput` runs immediately, no setup
2. **HTTP-First:** Everything via HTTP, no CLI/console required
3. **Progressive Enhancement:** Start simple, add complexity only when needed
4. **Persistent by Default:** Data survives restarts, can export/import
5. **JavaScript-Native:** Familiar runtime for web developers

---

## Next Steps

1. **Align on Vision:** Agree on use cases and scope
2. **Choose Data Layer:** Pick database strategy
3. **Fix Git State:** Clean up branch, commit v0.1 properly or start fresh
4. **Upgrade Go:** Move to modern Go version
5. **Design Schema:** Finalize data model
6. **Identify & Fix Bugs:** Test and repair existing issues
7. **Write Migration Guide:** If changing storage format

---

## Open Questions for Discussion

- Should we support multiple projects/databases in one hput instance?
- Is the `/dump` feature critical? (It's cool but complex)
- Do we need a web UI for management, or stay pure HTTP API?
- Should we add package/dependency management for JavaScript?
- What about WebSocket support for real-time apps?
- Should paths support query patterns like `/users/{id}`?

---

## Success Criteria

A successful v0.2 should:
- [ ] Store data persistently with proper schema
- [ ] Handle 1000+ paths efficiently
- [ ] Support listing/querying paths
- [ ] Run on modern Go with updated deps
- [ ] Have comprehensive tests
- [ ] Include clear migration path from v0.1
- [ ] Fix all known bugs
- [ ] Deploy easily (Docker? Binary release?)

---

## Summary & Recommendations

### Current State
- v0.1 works but is incomplete (no shared storage, no timestamps, old Go)
- Git state is broken (files deleted, code in v0.1/ folder)
- Go 1.21 is outdated (current: 1.25.5)
- Missing critical feature: JavaScript cannot access shared data storage

### Critical Decision Needed
**The shared storage architecture is the most important unresolved question.** We need to decide:

1. How JavaScript accesses data (API design)
2. What the security/isolation model should be

**Core Principle (MUST PRESERVE):**
> The path where you PUT is the SAME path where you GET.
> Programming the server = Using the server (unified concept).
> No artificial separation between "code paths" and "data paths".

**Recommended approach:** Subpath-Based Access Control
```javascript
// PUT JavaScript to /users
// This code can access ONLY /users/* (its descendants):

// Relative paths (automatically scoped to /users/*)
const user = await hput.get('123')           // Reads /users/123
await hput.put('123', {name: 'Alice'})       // Writes /users/123
const list = await hput.list('*')            // Lists /users/*
await hput.delete('456')                     // Deletes /users/456

// Absolute paths within scope also work
const user = await hput.get('/users/123')    // Reads /users/123 ✅

// CANNOT access outside scope
await hput.get('/records/123')               // ❌ Error: Outside /users/* scope
await hput.get('/posts/abc')                 // ❌ Error: Outside /users/* scope
```

**How it works:**
1. JavaScript at path `/X` can only access paths under `/X/*` (its subtree)
2. JavaScript CANNOT access paths outside its subtree (security boundary)
3. Relative paths are automatically prefixed with the code's path
4. No forced `/api/` or `/data/` conventions - users choose their structure
5. "Lego brick" simple: code at a path controls its descendant paths

**Example use cases:**
```javascript
// PUT JavaScript to /blog
// Can manage /blog/posts/*, /blog/comments/*, etc.

// PUT JavaScript to /users/admin
// Can manage /users/admin/* but NOT /users/regular/*

// PUT JavaScript to /
// Root-level code can access everything (careful!)
```

**Why this is best:**
- ✅ Preserves "PUT path = GET path" principle
- ✅ No artificial `/api/*` vs `/data/*` separation - user's choice
- ✅ Simple hierarchical security: tree-based permissions
- ✅ Relative paths make code cleaner and more portable
- ✅ Natural scoping: code controls its subtree
- ✅ "Lego brick" simple: each path is a mini-namespace
- ✅ Can add more complex permissions later (e.g., cross-path access)

### Recommended Next Steps

**Phase 1: Fix Environment**
1. Install Go 1.25.5 on M4 MacBook
2. Clean up git state (commit v0.1 or restore main)
3. Update go.mod to 1.25.5
4. Update dependencies (v8go, AWS SDK, bbolt)

**Phase 2: Design Shared Storage**
1. Implement subpath-based access control (code at `/X` can only access `/X/*`)
2. Add `hput` global object to JavaScript runtime with methods:
   - `hput.get(path)` - read data (relative or absolute within scope)
   - `hput.put(path, data)` - write data
   - `hput.list(pattern)` - list paths matching pattern
   - `hput.delete(path)` - delete data
3. Add relative path resolution (e.g., `'123'` → `/users/123` if code at `/users`)
4. Update `Runnable` struct to include metadata (timestamps, content-type)
5. Add scope checking logic to prevent out-of-subtree access

**Phase 3: Implement v0.2**
1. Add metadata storage
2. Implement JavaScript data access API
3. Write tests for new features
4. Update README with new capabilities
5. Create migration tool/script for v0.1 data

**Phase 4: Polish**
1. Fix identified bugs
2. Apply Go 1.25.5 syntax improvements
3. Add Docker support
4. Create examples showcasing shared storage

### Would You Like To...
- [ ] Start with Go installation and environment setup?
- [ ] Discuss the shared storage architecture more?
- [ ] Clean up the git state first?
- [ ] Review and identify bugs in v0.1?
- [ ] Explore alternative storage patterns?
