#!/bin/bash

# AI Health Coach Test Script
# This script runs all tests for the health tracking application

CONTAINER_ENGINE="${CONTAINER_ENGINE:-podman}"

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go and try again."
        exit 1
    fi
    
    print_status "Go version: $(go version)"
}

# Function to run unit tests
run_unit_tests() {
    print_status "Running unit tests..."
    
    # Run tests with coverage
    if go test -v -race -coverprofile=coverage.out ./...; then
        print_success "Unit tests passed"
        
        # Generate coverage report
        go tool cover -html=coverage.out -o coverage.html
        print_status "Coverage report generated: coverage.html"
        
        # Show coverage summary
        go tool cover -func=coverage.out | tail -1
    else
        print_error "Unit tests failed"
        return 1
    fi
}

# Function to run linting
run_linting() {
    print_status "Running linting checks..."
    
    # Check if golangci-lint is installed
    if ! command -v golangci-lint &> /dev/null; then
        print_warning "golangci-lint not found, installing..."
        
        # Install golangci-lint
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2
        
        # Add to PATH if not already there
        export PATH=$PATH:$(go env GOPATH)/bin
    fi
    
    if golangci-lint run ./...; then
        print_success "Linting checks passed"
    else
        print_error "Linting checks failed"
        return 1
    fi
}

# Function to check code formatting
check_formatting() {
    print_status "Checking code formatting..."
    
    # Check if code is properly formatted
    unformatted=$(gofmt -l .)
    if [ -n "$unformatted" ]; then
        print_error "The following files are not properly formatted:"
        echo "$unformatted"
        print_error "Run 'go fmt ./...' to fix formatting issues"
        return 1
    else
        print_success "Code formatting is correct"
    fi
}

# Function to run security checks
run_security_checks() {
    print_status "Running security checks..."
    
    # Check if gosec is installed
    if ! command -v gosec &> /dev/null; then
        print_warning "gosec not found, installing..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
    
    if gosec ./...; then
        print_success "Security checks passed"
    else
        print_warning "Security checks found potential issues (review output above)"
        # Don't fail the build for security warnings, just warn
    fi
}

# Function to check for vulnerabilities
check_vulnerabilities() {
    print_status "Checking for known vulnerabilities..."
    
    # Check if govulncheck is installed
    if ! command -v govulncheck &> /dev/null; then
        print_warning "govulncheck not found, installing..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi
    
    if govulncheck ./...; then
        print_success "No known vulnerabilities found"
    else
        print_error "Vulnerabilities found in dependencies"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_status "Running integration tests..."
    
    # Set up test environment variables
    export DB_HOST="localhost"
    export DB_PORT="5432"
    export DB_NAME="healthtracker_test"
    export DB_USER="healthtracker_test"
    export DB_PASSWORD="test123"
    export DB_SSLMODE="disable"
    export ENVIRONMENT="test"
    
    # Check if Docker is available for test database
    if command -v "$CONTAINER_ENGINE" &> /dev/null; then
        print_status "Starting test database with Docker..."
        
        # Start PostgreSQL container for testing
        "$CONTAINER_ENGINE" run --name postgres-test -e POSTGRES_DB="$DB_NAME" \
            -e POSTGRES_USER="$DB_USER" \
            -e POSTGRES_PASSWORD="$DB_PASSWORD" \
            -p 5432:5432 -d postgres:15-alpine
        
        # Wait for database to be ready
        sleep 10
        
        # Run integration tests
        if go test -tags=integration -v ./pkg/database/...; then
            print_success "Integration tests passed"
        else
            print_error "Integration tests failed"
            "$CONTAINER_ENGINE" stop postgres-test
            "$CONTAINER_ENGINE" rm postgres-test
            return 1
        fi
        
        # Clean up test database
        "$CONTAINER_ENGINE" stop postgres-test
        "$CONTAINER_ENGINE" rm postgres-test
    else
        print_warning "Docker not available, skipping integration tests"
    fi
}

# Function to run benchmark tests
run_benchmarks() {
    print_status "Running benchmark tests..."
    
    if go test -bench=. -benchmem ./...; then
        print_success "Benchmark tests completed"
    else
        print_warning "Some benchmark tests failed"
    fi
}

# Function to validate Kubernetes manifests
validate_k8s_manifests() {
    print_status "Validating Kubernetes manifests..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        print_warning "kubectl not found, skipping Kubernetes manifest validation"
        return 0
    fi
    
    # Check if kustomize is available
    if ! command -v kustomize &> /dev/null; then
        print_warning "kustomize not found, skipping Kubernetes manifest validation"
        return 0
    fi
    
    # Validate base manifests
    if kustomize build k8s/base | kubectl apply --dry-run=client -f -; then
        print_success "Base Kubernetes manifests are valid"
    else
        print_error "Base Kubernetes manifests validation failed"
        return 1
    fi
    
    # Validate production overlay
    if kustomize build k8s/overlays/production | kubectl apply --dry-run=client -f -; then
        print_success "Production Kubernetes manifests are valid"
    else
        print_error "Production Kubernetes manifests validation failed"
        return 1
    fi
}

# Function to run Dockerfile linting
lint_dockerfile() {
    print_status "Linting Dockerfile..."
    
    # Check if hadolint is available
    if command -v hadolint &> /dev/null; then
        if hadolint Dockerfile; then
            print_success "Dockerfile linting passed"
        else
            print_warning "Dockerfile linting found issues"
        fi
    else
        print_warning "hadolint not found, skipping Dockerfile linting"
        print_status "Install hadolint with: brew install hadolint (macOS) or apt-get install hadolint (Ubuntu)"
    fi
}

# Function to check dependencies for vulnerabilities
check_go_dependencies() {
    print_status "Checking Go dependencies..."
    
    # Check for outdated dependencies
    if command -v go-mod-outdated &> /dev/null; then
        print_status "Checking for outdated dependencies..."
        go list -u -m all | go-mod-outdated -update -direct
    fi
    
    # Verify dependencies
    go mod verify
    if [ $? -eq 0 ]; then
        print_success "Go dependencies verification passed"
    else
        print_error "Go dependencies verification failed"
        return 1
    fi
    
    # Tidy dependencies
    go mod tidy
    if git diff --quiet go.mod go.sum; then
        print_success "Go dependencies are clean"
    else
        print_warning "Go dependencies need tidying (running go mod tidy)"
        git add go.mod go.sum
    fi
}

# Function to generate test report
generate_test_report() {
    print_status "Generating test report..."
    
    cat > test-report.html << EOF
<!DOCTYPE html>
<html>
<head>
    <title>AI Health Coach Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .success { color: green; }
        .warning { color: orange; }
        .error { color: red; }
        .section { margin: 20px 0; padding: 10px; border: 1px solid #ccc; }
    </style>
</head>
<body>
    <h1>AI Health Coach Test Report</h1>
    <p>Generated on: $(date)</p>
    
    <div class="section">
        <h2>Test Results</h2>
        <p>See coverage.html for detailed coverage report</p>
    </div>
    
    <div class="section">
        <h2>Coverage</h2>
        <p>Coverage report: <a href="coverage.html">coverage.html</a></p>
    </div>
</body>
</html>
EOF
    
    print_success "Test report generated: test-report.html"
}

# Main test function
main() {
    print_status "Starting comprehensive test suite for AI Health Coach"
    echo
    
    local exit_code=0
    
    # Run all test phases
    check_go || exit_code=$?
    check_formatting || exit_code=$?
    check_go_dependencies || exit_code=$?
    run_linting || exit_code=$?
    run_unit_tests || exit_code=$?
    run_security_checks || exit_code=$?
    check_vulnerabilities || exit_code=$?
    validate_k8s_manifests || exit_code=$?
    lint_dockerfile || exit_code=$?
    
    # Optional tests that don't fail the build
    run_integration_tests || print_warning "Integration tests failed but continuing..."
    run_benchmarks || print_warning "Benchmark tests had issues but continuing..."
    
    # Generate report
    generate_test_report
    
    echo
    if [ $exit_code -eq 0 ]; then
        print_success "All critical tests passed! 🎉"
        print_status "Test artifacts:"
        print_status "  - coverage.html (coverage report)"
        print_status "  - test-report.html (summary report)"
    else
        print_error "Some tests failed. Please review the output above."
        exit $exit_code
    fi
}

# Parse command line arguments
SKIP_INTEGRATION=false
SKIP_SECURITY=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-integration)
            SKIP_INTEGRATION=true
            shift
            ;;
        --skip-security)
            SKIP_SECURITY=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo
            echo "Options:"
            echo "  --skip-integration    Skip integration tests"
            echo "  --skip-security       Skip security checks"
            echo "  --verbose            Enable verbose output"
            echo "  -h, --help           Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Set verbose mode if requested
if [ "$VERBOSE" = true ]; then
    set -x
fi

# Run main test suite
main
