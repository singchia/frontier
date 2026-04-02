#!/bin/bash

# Frontier Comprehensive Test Suite Runner — see `test/run_tests.sh -h`

# Benchmark -benchtime defaults
: "${BENCH_TIME_EACH:=3s}"
: "${BENCH_TIME_ALL:=10s}"
# go test -timeout: applies to the whole process (default: ~10m, too short for bench=. + long benchtime)
: "${BENCH_GO_TEST_TIMEOUT:=30m}"

OUTPUT_FILE=""
run_unit=false
run_bench=false
run_e2e=false
run_security=false
run_race=false
run_cover=false
any_category=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -o|--output)
            if [[ -z "${2:-}" ]]; then
                echo "Error: $1 requires a file path" >&2
                exit 1
            fi
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -h|--help)
            cat <<'EOF'
Usage: test/run_tests.sh [options] [--category ...]

Options:
  -o, --output FILE   Write full output to FILE (overwrite; also shown on terminal)
  -h, --help          Show this help

Categories (combine multiple; omit all to run everything):
  --unit       Unit tests (exchange): go test ./... -short
  --bench      Benchmarks under test/bench
  --e2e        End-to-end tests under test/e2e
  --security   Security tests under test/security (race, fuzz)
  --race       Race detector on unit tests
  --cover      Coverage (coverage.out, coverage.html)
  --all        Explicitly run all categories

Examples:
  test/run_tests.sh --unit
  test/run_tests.sh --e2e --security
  test/run_tests.sh -o run.log --bench

Env (benchmark section only):
  BENCH_TIME_EACH        Per-benchmark -benchtime (default: 3s)
  BENCH_TIME_ALL         Final bench=. -benchtime (default: 10s)
  BENCH_GO_TEST_TIMEOUT  go test -timeout for benchmarks (default: 30m; 0 = no limit)
EOF
            exit 0
            ;;
        --unit)
            run_unit=true
            any_category=true
            shift
            ;;
        --bench)
            run_bench=true
            any_category=true
            shift
            ;;
        --e2e)
            run_e2e=true
            any_category=true
            shift
            ;;
        --security)
            run_security=true
            any_category=true
            shift
            ;;
        --race)
            run_race=true
            any_category=true
            shift
            ;;
        --cover|--coverage)
            run_cover=true
            any_category=true
            shift
            ;;
        --all)
            run_unit=true
            run_bench=true
            run_e2e=true
            run_security=true
            run_race=true
            run_cover=true
            any_category=true
            shift
            ;;
        *)
            echo "Unknown option: $1 (try -h)" >&2
            exit 1
            ;;
    esac
done

if ! $any_category; then
    run_unit=true
    run_bench=true
    run_e2e=true
    run_security=true
    run_race=true
    run_cover=true
fi

# Resolve relative log path to invocation cwd (before cd to project root)
if [[ -n "$OUTPUT_FILE" ]]; then
    if [[ "$OUTPUT_FILE" != /* ]]; then
        OUTPUT_FILE="$(pwd)/$OUTPUT_FILE"
    fi
    mkdir -p "$(dirname "$OUTPUT_FILE")"
    exec > >(tee "$OUTPUT_FILE") 2>&1
fi

set -e

echo "==================================="
echo "Frontier Comprehensive Test Suite"
echo "==================================="
if [[ -n "$OUTPUT_FILE" ]]; then
    echo "Full output also logged to: $OUTPUT_FILE"
fi
echo "Categories:"
$run_unit       && echo "  - unit"
$run_bench      && echo "  - bench"
$run_e2e        && echo "  - e2e"
$run_security   && echo "  - security (race, fuzz)"
$run_race       && echo "  - race"
$run_cover      && echo "  - cover"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print section headers
print_header() {
    echo ""
    echo -e "${YELLOW}===================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}===================================${NC}"
    echo ""
}

# Change to project root
cd "$(dirname "$0")/.." 

# Install dependencies
echo "Installing dependencies..."
go mod download

if $run_unit; then
    print_header "Running Unit Tests (Exchange)"
    go test -v ./pkg/frontier/exchange/... -short -count=1 2>&1 | tail -50 || true
fi

if $run_bench; then
    print_header "BENCHMARK TESTS"

    echo "Running RPC Call Benchmarks..."
    go test -timeout="${BENCH_GO_TEST_TIMEOUT}" -bench=BenchmarkEdgeCall -benchmem -benchtime="${BENCH_TIME_EACH}" ./test/bench/... 2>&1 || true

    echo ""
    echo "Running Service Call Benchmarks..."
    go test -timeout="${BENCH_GO_TEST_TIMEOUT}" -bench=BenchmarkServiceCall -benchmem -benchtime="${BENCH_TIME_EACH}" ./test/bench/... 2>&1 || true

    echo ""
    echo "Running Message Publishing Benchmarks..."
    go test -timeout="${BENCH_GO_TEST_TIMEOUT}" -bench=BenchmarkEdgePublish -benchmem -benchtime="${BENCH_TIME_EACH}" ./test/bench/... 2>&1 || true

    echo ""
    echo "Running Stream Open Benchmarks..."
    go test -timeout="${BENCH_GO_TEST_TIMEOUT}" -bench=BenchmarkEdgeOpen -benchmem -benchtime="${BENCH_TIME_EACH}" ./test/bench/... 2>&1 || true

    echo ""
    echo "Running Edge Connect/Disconnect Benchmarks..."
    go test -timeout="${BENCH_GO_TEST_TIMEOUT}" -bench=BenchmarkEdgeConnect -benchmem -benchtime="${BENCH_TIME_EACH}" ./test/bench/... 2>&1 || true

    print_header "Running All Benchmarks (${BENCH_TIME_ALL} each)..."
    go test -timeout="${BENCH_GO_TEST_TIMEOUT}" -bench=. -benchmem -benchtime="${BENCH_TIME_ALL}" ./test/bench/... 2>&1 || true
fi

if $run_e2e; then
    print_header "E2E INTEGRATION TESTS"

    echo "Running Connection Tests..."
    go test -v -run TestConn ./test/e2e/... -count=1 2>&1 | tail -100 || true

    echo ""
    echo "Running RPC Tests..."
    go test -v -run TestRPC ./test/e2e/... -count=1 2>&1 | tail -100 || true

    echo ""
    echo "Running Message Tests..."
    go test -v -run TestMessage ./test/e2e/... -count=1 2>&1 | tail -100 || true

    echo ""
    echo "Running Stream Tests..."
    go test -v -run TestStream ./test/e2e/... -count=1 2>&1 | tail -100 || true

    echo ""
    echo "Running Resource Cleanup Tests..."
    go test -v -run TestResourceCleanup ./test/e2e/... -count=1 2>&1 | tail -50 || true

    print_header "Running All E2E Tests..."
    go test -v ./test/e2e/... -count=1 -timeout=10m 2>&1 | tail -100 || true
fi

if $run_security; then
    print_header "SECURITY TESTS (Race & Fuzz)"

    echo "Running Race Condition Tests..."
    go test -v -race ./test/security/... -count=1 -timeout=5m 2>&1 | tail -100 || true

    echo ""
    echo "Running Fuzz Tests..."
    if go version | grep -qE 'go1\.(1[89]|2[0-9]|[3-9][0-9])'; then
        go test -v -run TestFuzz ./test/security/... -count=1 2>&1 | tail -50 || true
        echo ""
        echo "Running native fuzz (30 seconds)..."
        go test -fuzz=Fuzz -fuzztime=30s ./test/security/... 2>&1 || true
    else
        echo "Go version doesn't support fuzzing natively (requires Go 1.18+), skipping..."
    fi

    print_header "Running All Security Tests..."
    go test -v ./test/security/... -count=1 -timeout=10m 2>&1 | tail -100 || true
fi

if $run_race; then
    print_header "RACE DETECTION TESTS"

    echo "Running unit tests with race detector..."
    go test -race -short ./pkg/frontier/exchange/... 2>&1 | tail -100 || true
fi

if $run_cover; then
    print_header "CODE COVERAGE"

    echo "Generating coverage report..."
    go test -coverprofile=coverage.out ./pkg/frontier/... ./test/e2e/... 2>&1 || true
    go tool cover -func=coverage.out | tail -30 || true

    if command -v go &> /dev/null; then
        go tool cover -html=coverage.out -o coverage.html 2>&1 || true
        echo "Coverage report generated: coverage.html"
    fi
fi

print_header "TEST SUMMARY"

echo -e "${GREEN}Test execution completed!${NC}"
echo ""
echo "Test categories executed:"
$run_unit       && echo "  - Unit Tests (Exchange)"
$run_bench      && echo "  - Benchmark Tests"
$run_e2e        && echo "  - E2E Integration Tests"
$run_security   && echo "  - Security Tests (Race, Fuzz)"
$run_race       && echo "  - Race Detection Tests"
$run_cover      && echo "  - Code Coverage"
echo ""
echo "Check the output above for any failures or issues."
echo ""
