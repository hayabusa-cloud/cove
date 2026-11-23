# Measuring cove

Run the benchmarks with Go 1.27+ on the machine and dependency versions used by your application. Record `go version`, `go env GOOS GOARCH GOEXPERIMENT`, `go list -m all`, CPU details, and `GOMAXPROCS` with each measurement.

The [Benchmarks workflow](.github/workflows/benchmarks.yml) can also be run manually on GitHub. Choose the benchmark filter, duration and repetition count; the job records its environment and raw measurements in the logs. Use the commands below to measure on your deployment machine.

```bash
go test -timeout=120s ./...
go test -run '^$' -bench '^Benchmark(Temporal|ForcingTransition)' -benchmem -benchtime=200ms -count=10 -timeout=120s . > benchmarks.txt
```

Use `-bench` to select a workload and adjust `-benchtime`, `-count`, and `GOMAXPROCS` for your environment. Keep these settings and dependency versions identical when comparing implementations. Repeat measurements under comparable load; use `benchstat` if available to summarize variation.

| Workload | Measurement boundary |
|----------|----------------------|
| `BenchmarkObserve` | A retained value/context pair with changing fields |
| `BenchmarkNeed*`, `BenchmarkReflectReqEval`, `BenchmarkReifyReqEval`, `BenchmarkCheckRuleExpr`, `BenchmarkCheckedExprIntoView` | Preconstructed requirements and rules across negative, zero, odd, even and boundary inputs; decisions and returned views are retained |
| `BenchmarkStepExprWith`, `BenchmarkResumeWith` | Complete expression activations with a fresh suspension and retained completed result; program construction is excluded |
| `BenchmarkTemporalObservation` | The same eligibility and generation check on changing contexts, using direct fields, `Observe`/`Need`, or a prebuilt expression with a direct or projected generation predicate |
| `BenchmarkTemporalPullbackSetup` | Construction and retention of `Pullback` and `ExprPullback`; setup costs are included |
| `BenchmarkTemporalResume` | A complete guarded activation, including a fresh kont suspension and terminal resumption |
| `BenchmarkTemporalSuccessor` | A complete activation that yields and consumes a successor; immutable program construction is excluded |
| `BenchmarkForcingTransition` | Direct and cove edge/endpoint checks across changing worlds; requirement construction is excluded |

Compare corresponding direct and cove cases within a workload. Requirement construction is excluded from observation measurements; constructing requirements per event makes that cost part of your event path. Complete activations include kont's continuation and frame costs, so they do not isolate the context wrapper.

The Makefile provides `make check` for format, module-drift and vet checks, `make test` for race and coverage checks, and `make bench` for the full suite. `make check` reports necessary changes without rewriting source or module files. Use the Go commands above when selecting workloads or changing run settings.

Inspect allocation counts alongside timing. Rounded `0 allocs/op` does not establish absence of first-use allocation. For a concrete instantiation, inspect escape diagnostics and emitted code as well:

```bash
go test -c -o cove.test -gcflags='-m=2 -S' . 2> codegen.txt
```

The log contains escape diagnostics and compiler assembly for the tested package. Follow calls that remain in the measured loop and inspect the relevant dependencies separately; account for frames, copies, and indirect dispatch. These microbenchmarks measure local operations; they do not establish application tail latency, deadline compliance, or idle CPU behavior.
