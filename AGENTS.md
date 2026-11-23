# cove

Explicit ambient context, requirements, contextual commands, and kont suspension observations. Keep the package small and policy-free; event acquisition, scheduling, retries, and I/O resource retirement belong to its consumers.

## Contracts

- `View` is a value/context pair. Its algebraic laws assume pure callbacks and stable context; copies share any referenced storage.
- `Req` treats nil as true. `ReqExpr` has a true zero value; `ExprAtom` and `ReifyReq` require non-nil predicates. Composite constructors retain multiple operands supplied as a slice.
- Requirement composition evaluates left to right with short circuiting. Bind requirements before the hot path when construction is cold; keep callbacks, context, and borrowed operand storage stable during evaluation.
- Copying a suspension view does not duplicate the right to resume. Refusal preserves the original handle; resume or discard each held suspension once and retain only the returned successor.
- Completion is identified by a nil suspension. Completed values, including nil, are valid results. `ResumeWith` maps context after advancement, including terminal return.
- Kripke helpers check caller-supplied witnesses. `StepIndex` and `Later` express logical approximation; concrete edge checks establish no global law, freshness, deadline, or external resource release.

Caller code owns concurrency control and the protected interval between a requirement check and its authorized action. Avoid adding synchronization, copying, or validation that duplicates that contract.

## Verification

Use Go 1.27+ and black-box tests in `package cove_test`.

```bash
make check
make test
```

Cross-build the CI targets with `GOOS=linux GOARCH=<arch> go build ./...`: amd64, arm64, riscv64, and loong64. Keep the five READMEs consistent with GoDoc and executable examples. Follow [BENCHMARKS.md](BENCHMARKS.md) for comparable workloads, allocation checks, and cost boundaries.
