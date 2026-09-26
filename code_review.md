# Glib — Code Review: Refactoring & Improvement Opportunities

## Summary

The codebase is well-structured and idiomatic Go. The main concerns are **code duplication** across the three scan paths, a few **design inconsistencies**, some **performance surprises**, and several **simplification opportunities** using stdlib or modern Go idioms.

---

## 🔴 High Priority

### 1. Duplicated "Second Pass" Handler Scanning Logic (3 copies)

The logic that groups files by package, ensures type resolution files, then scans handlers is **copy-pasted verbatim** across three files:

| File | Function |
|---|---|
| [`scanner.go`](file:///home/azizndao/repos/web/glib/internal/scanner/scanner.go#L193-L249) | `Scan()` — inline second pass |
| [`parallel.go`](file:///home/azizndao/repos/web/glib/internal/scanner/parallel.go#L156-L211) | `scanHandlersForControllers()` |
| [`incremental.go`](file:///home/azizndao/repos/web/glib/internal/scanner/incremental.go#L100-L153) | inline second pass |

**Both `parallel.go` and `incremental.go` already call `scanHandlersForControllers()`, but `scanner.go` has its own inlined version of the same logic.**

**Fix:** Extract a unified `runHandlerScanPass(project *Project, fileMap map[string]*ast.File) error` and call it from all three paths.

---

### 2. `scanLocaleFiles` Called Twice in `ScanIncremental`

In [`incremental.go`](file:///home/azizndao/repos/web/glib/internal/scanner/incremental.go):

```go
// Line 155-164: first call (inline)
if s.i18nEnabled && s.i18nLocaleDir != "" {
    localesPath := ...
    if _, err := os.Stat(localesPath); err == nil {
        localeFiles, err := ScanLocales(localesPath)
        ...
    }
}

// Line 176-178: second call (via helper)
if err := s.scanLocaleFiles(project); err != nil { ...}
```

The locale files are scanned **twice** per incremental scan. The first inline block is a leftover. One of them can simply be deleted.

---

### 3. Dead-Code Duplication in `codegen_shared.go`

```go
// Lines 103-107 in codegen_shared.go
if isIncremental {
    if opts.ShowProgress {
        fmt.Println(ui.Infof("Incremental scan (%d files)...", ...))
    } else {
        fmt.Println(ui.Infof("Incremental scan (%d files)...", ...)) // identical!
    }
}
```

Both branches of the `if opts.ShowProgress` are identical. Remove the branch.

Similarly, lines 128–137 have overlapping conditions that can be collapsed:

```go
outputMode := OutputModeDetailed
if !opts.ShowProgress {
    outputMode = OutputModeCompact
}
if opts.Verbose {
    printScanStats(...)
} else if !isIncremental && opts.ShowProgress { ... }
else if !isIncremental { ... }  // same body as previous
```

---

### 4. `Scanner` Struct Has Mutable Per-File State (Thread-Safety Hazard)

[`scanner.go`](file:///home/azizndao/repos/web/glib/internal/scanner/scanner.go#L26-L46):

```go
type Scanner struct {
    currentPackageName string    // mutated per-file
    currentPackagePath string    // mutated per-file
    currentImports     map[string]string  // mutated per-file
    currentFile        *ast.File          // mutated per-file
    typeSpecs          map[string]*ast.TypeSpec  // mutated per-file
    mu                 sync.Mutex
    ...
}
```

These fields represent **ephemeral per-file context** stored on the Scanner itself, which forces a mutex on every `scanFile()` call. The parallel worker pool must lock on every scan. This is the main bottleneck for parallel mode.

**Fix:** Extract into a `scanContext` struct passed as an argument:
```go
type scanContext struct {
    packageName string
    packagePath string
    imports     map[string]string
    typeSpecs   map[string]*ast.TypeSpec
}
```
`scanFile`, `parseType`, `parseHandlerSignature`, etc. would then take `*scanContext`. This removes the global mutex from the hot path and makes parallel scanning truly parallel.

---

## 🟡 Medium Priority

### 5. `sortMiddleware` Uses Bubble Sort

In [`parsers.go`](file:///home/azizndao/repos/web/glib/internal/generator/parsers.go#L251-L265):

```go
// Simple bubble sort (fine for small lists)
for i := range sorted {
    for j := i + 1; j < len(sorted); j++ {
        if compareMiddleware(sorted[i], sorted[j]) > 0 {
            sorted[i], sorted[j] = sorted[j], sorted[i]
        }
    }
}
```

The comment says "fine for small lists" but this is called **per handler** during code generation. Replace with `sort.Slice` or `slices.SortFunc` (already imported in the file) using `compareMiddleware`:

```go
slices.SortFunc(sorted, compareMiddleware)
```

---

### 6. `joinStrings` Duplicates `strings.Join`

[`controllers.go`](file:///home/azizndao/repos/web/glib/internal/scanner/controllers.go#L295-L307):

```go
func joinStrings(strs []string, sep string) string {
    // 12 lines of manual builder code
}
```

This is a re-implementation of `strings.Join`. Replace with:
```go
typeInfo.FullName = baseType.FullName + "[" + strings.Join(paramNames, ", ") + "]"
```

---

### 7. Config Provider Name Uses Magic String Sentinel (`__config_...__`)

In [`di.go`](file:///home/azizndao/repos/web/glib/internal/generator/di.go#L239-L252):

```go
Name: "__config_" + cfg.Name + "__",   // synthetic sentinel

// Later:
if strings.HasPrefix(prov.Name, "__config_") {
    continue
}
// And:
configName := strings.TrimSuffix(strings.TrimPrefix(prov.Name, "__config_"), "__")
```

This leaks a naming convention as a fragile string sentinel. A cleaner approach: add a boolean field `IsConfig bool` to `scanner.Provider` (or to `ProviderData`), which is already partially done in `ProviderData.IsConfig`. The `IsConfig` flag on `ProviderData` is set correctly, but the upstream filtering still uses the magic prefix. Unify to use the flag.

---

### 8. `getMiddlewareOrder` Parses Its Own Previously-Built String

In [`routes.go`](file:///home/azizndao/repos/web/glib/internal/generator/routes.go#L201-L214):

```go
func (g *Generator) getMiddlewareOrder(middlewareName string) int {
    // Input: "app.middleware.AuthMiddleware"
    parts := strings.Split(middlewareName, ".")
    name := strings.TrimSuffix(parts[2], "Middleware")
    // Then scans all middleware to find by name
    for _, mw := range g.project.Middleware {
        if capitalize(mw.Name) == name { return mw.Order }
    }
}
```

The string `"app.middleware.AuthMiddleware"` was **generated by the same function** a few lines above it. Instead of building the string and then parsing it back, pass `*scanner.Middleware` directly. Both `getControllerMiddleware` and `getTagGroupMiddleware` could return `[]*scanner.Middleware` instead of `[]string`, and the string formatting be done at template rendering time.

---

### 9. `isFileCached` Computes the Hash Twice

In [`scanner/cache.go`](file:///home/azizndao/repos/web/glib/internal/scanner/cache.go#L94-L126):

```go
func (c *FileCache) isFileCached(filePath string) (bool, string, error) {
    info, _ := os.Stat(filePath)
    
    // First hash computation (quick path)
    if entry, ok := c.Get(filePath, info.ModTime()); ok {
        hash, _ := computeFileHash(filePath)   // hash #1
        if entry.Hash == hash { return true, hash, nil }
    }
    
    // Second hash computation (slow path)
    hash, _ := computeFileHash(filePath)       // hash #2
    ...
}
```

When the mod-time check passes but the hash check fails (a rare race), the hash is computed twice. Minor, but easily fixed with a single `computeFileHash` call at the top.

---

### 10. `findProviderForType` Does Three Linear Scans Per Call

In [`di.go`](file:///home/azizndao/repos/web/glib/internal/generator/di.go#L299-L331):

```go
func (g *Generator) findProviderForType(typeInfo *scanner.TypeInfo) string {
    // Scan 1: configs (O(n))
    for _, cfg := range g.project.Configs { ... }
    // Scan 2: providers (O(n))
    for _, prov := range g.project.Providers { ... }
}
```

This is called for **every field of every controller and every dependency of every provider**. With N providers and M controllers each with K fields, this is O(N×M×K). A simple `map[string]string` (fullName → fieldName) built once in `generateDI()` would reduce this to O(1) per lookup.

---

### 11. `Provider` Has Both `Name` and `FunctionName` That Are Always Equal

In [`providers.go`](file:///home/azizndao/repos/web/glib/internal/scanner/providers.go#L52-L64):

```go
provider := &Provider{
    Name:         funcDecl.Name.Name,
    FunctionName: funcDecl.Name.Name,  // always the same
    ...
}
```

And in [`models.go`](file:///home/azizndao/repos/web/glib/internal/scanner/models.go#L235-L246) both fields exist. These are always the same value. Consider removing one or documenting when they differ.

---

## 🟢 Low Priority / Style

### 12. `parseHandlerSignature` Validates Then Ignores Return Constraint for Raw HTTP

```go
// handlers.go line 73-75
if len(returns) > 0 {
    return nil, fmt.Errorf("raw HTTP handler must not return any value, got %d return values", len(returns))
}
```

But raw HTTP handlers can return nothing or optionally nothing — the `void` constraint is enforced correctly. However, the error message says "must not return any value" which is slightly confusing since `(w, r)` → nothing is the only valid shape. The message is fine, just could mention "use `(T, error)` or `error` patterns for typed responses".

---

### 13. `strings.SplitSeq` Used Inconsistently

In `routes.go` and `validator.go`, `strings.SplitSeq` (Go 1.24+) is used correctly. But in `annotations.go`, `strings.FieldsSeq` is used. These are consistent with Go 1.24+ iterator usage, but double-check minimum supported Go version since `go.mod` says `go 1.27` which is fine.

---

### 14. `parallel.go` Only Collects First Error

```go
if len(scanErrors) > 0 {
    return nil, fmt.Errorf("scan errors: %v", scanErrors[0])  // only first
}
```

When parallel scanning hits multiple errors (e.g., malformed files), only the first is returned. Consider using `errors.Join(scanErrors...)` (Go 1.20+) to surface all failures at once.

---

### 15. `validate.go` (CLI) Wraps `validator.go` But Duplicates Formatting Logic

The CLI `validate` command in `internal/cli/validate.go` and `PerformCodeGeneration` in `codegen_shared.go` both format and print validation errors differently. A shared `formatValidationErrors(errors []*validator.ValidationError) string` helper would eliminate this.

---

## Refactoring Priority Order

```
1. Extract unified handler scan pass (eliminates ~100 lines of duplication)
2. Fix double locale scan in ScanIncremental (bug)
3. Fix duplicate log line in codegen_shared.go (bug)
4. Extract per-file scanContext (enables true parallel scanning)
5. Replace findProviderForType linear scan with a map
6. Replace bubble sort with slices.SortFunc
7. Replace joinStrings with strings.Join
8. Remove magic __config_ sentinel
9. Pass *Middleware directly instead of building+parsing strings
10. Fix double hash computation in isFileCached
```
