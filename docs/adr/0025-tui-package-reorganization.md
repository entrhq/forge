# 0025. TUI Package Reorganization (Interface-Driven Design)

**Status:** Implemented
**Date:** 2024-06-17
**Deciders:** Developer, Architect
**Technical Story:** Refactoring the monolithic `pkg/executor/tui` package into maintainable subpackages.

---

## Context

The `pkg/executor/tui` package currently contains 33 files in a flat structure. This violates the Single Responsibility Principle (SRP) and makes the codebase difficult to navigate and maintain. All components (overlays, commands, event handling) are tightly coupled to a central `model` struct.

### Problem Statement

We need to split the monolithic TUI package into focused subpackages (e.g., `overlay`, `syntax`, `commands`) to improve modularity and maintainability.

However, a previous attempt to do this failed because of **circular dependencies**. The extracted subpackages tried to import the `model` from the root package to access state and methods, while the root package tried to import the subpackages to initialize them.

### Goals

- Break `pkg/executor/tui` into focused, cohesive subpackages.
- Eliminate circular dependencies between packages.
- Enforce Dependency Inversion (depend on abstractions, not concretions).
- Improve testability of individual components.

### Non-Goals

- Rewriting the entire UI framework (we stick with Bubble Tea).
- Changing the external API of the `Executor`.

---

## Decision Drivers

* **Maintainability:** 33 files in one folder is unmanageable.
* **Go Package Design:** Go forbids import cycles.
* **Testing:** Tightly coupled components are hard to test in isolation.
* **Clean Architecture:** We want to respect SRP and Dependency Inversion.

---

## Considered Options

### Option 1: Flat Structure with Naming Convention

**Description:** Keep all files in one package but use prefixes (e.g., `overlay_help.go`) to group them.
**Pros:**
- Zero risk of import cycles.
- Simple to implement.
**Cons:**
- Does not solve the SRP violation at the package level.
- Still one massive namespace.
- "Poor man's modules".

### Option 2: Shared "Context" Package

**Description:** Move the entire `model` struct to a `shared` or `context` package that everyone imports.
**Pros:**
- Solves circular dependencies easily.
**Cons:**
- Creates a "God Object" in a separate package.
- Exposes internal state that should be private.
- Violates encapsulation.

### Option 3: Interface-Driven Design (Dependency Inversion)

**Description:** Define interfaces in a leaf package (`types`) that describe the capabilities components need. Subpackages depend on these interfaces, not the concrete `model`. The root `model` implements these interfaces.

**Structure:**
```
tui/
  ├── types/ (Interfaces only)
  ├── overlay/ (Depends on types)
  ├── functionality... (Depends on types)
  └── model.go (Root - Depends on types & overlay)
```

**Pros:**
- True decoupling.
- Enforces strict contracts between layers.
- Solves import cycles architecturally (DAG).
- Highly testable (can mock interfaces).
**Cons:**
- More boilerplate (need to define interfaces).
- Higher initial refactoring effort.

---

## Decision

**Chosen Option:** Option 3 - Interface-Driven Design (Dependency Inversion)

### Rationale

This is the only option that achieves **true package separation** while adhering to Go's dependency rules and Clean Architecture principles. By inverting the dependencies, we ensure that low-level components (overlays) do not know about the high-level orchestrator (model), but only about the contracts (interfaces) they require to function.

---

## Consequences

### Positive

- **Decoupling:** Subpackages like `overlay` and `syntax` will have zero knowledge of the main application logic, making them reusable and testable.
- **Organization:** The codebase will be organized into logical, focused packages (`syntax`, `approval`, `commands`, `overlay`).
- **Scalability:** New features can be added as new packages without touching the core.

### Negative

- **Boilerplate:** We must define `StateProvider` and `ActionHandler` interfaces.
- **Refactoring Cost:** Every method signature that currently takes `*model` needs to be updated to take an interface.

### Neutral

- **Indirection:** Navigation will sometimes require jumping to interface definitions before seeing implementations.

---

## Implementation

### Migration Path

1.  **Foundation:** Create `pkg/executor/tui/types` and move `AgentEvent`, constants, and define `StateProvider`/`ActionHandler` interfaces.
2.  **Independent Layers:** Extract `tui/syntax` and `tui/approval` (simplest dependencies).
3.  **Component Refactoring:**
    - Take one overlay (e.g., `HelpOverlay`).
    - Change its `Update` method signature to accept `types.StateProvider` instead of `*model`.
    - Move it to `tui/overlay`.
    - Repeat for all overlays.
4.  **Integration:** Update `tui/model.go` to implicitly implement the new interfaces.

### Timeline

Immediate execution.

---

## Validation

### Success Metrics

- **Pass:** `go build ./pkg/executor/tui/...` succeeds (no import cycles).
- **Pass:** All tests pass.
- **Metric:** Number of files in root `pkg/executor/tui` reduces from 33 to <10.

### Monitoring

- Go compiler will enforce the acyclic dependency graph.

---

## References

- [Dependency Inversion Principle](https://en.wikipedia.org/wiki/Dependency_inversion_principle)
- [Go Package Layout](https://github.com/golang-standards/project-layout)