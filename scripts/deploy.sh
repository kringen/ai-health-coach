#!/bin/bash

# AI Health Coach Deployment Script
# This script builds, containerizes, and deploys the health tracking application
#
# Usage: ./deploy.sh [ENVIRONMENT] [ACTION] [REGISTRY] [TAG] [HOSTNAME]
# Example: ./deploy.sh development kringen v0.0.1 tracker.kringen.io

set -e  # Exit on any error

# Handle help flag first
for arg in "$@"; do
    if [[ "$arg" == "-h" ]] || [[ "$arg" == "--help" ]]; then
        echo "Usage: $0 [ENVIRONMENT] [ACTION] [REGISTRY] [TAG] [HOSTNAME] [OPTIONS]"
        echo
        echo "Positional Arguments:"
        echo "  ENVIRONMENT    Deployment environment (default: development)"
        echo "  ACTION         Action to perform: apply, deploy, delete, destroy, status (default: apply)"
        echo "  REGISTRY       Container registry URL (optional)"
        echo "  TAG            Image tag (default: latest)"
        echo "  HOSTNAME       Hostname for ingress (optional)"
        echo
        echo "Options:"
        echo "  --skip-tests   Skip running tests"
        echo "  --skip-push    Skip pushing Docker image"
        echo "  -h, --help     Show this help message"
        echo
        echo "Examples:"
        echo "  $0                                    # Deploy to development with defaults"
        echo "  $0 production apply                   # Deploy to production"
        echo "  $0 development kringen v0.0.1 tracker.kringen.io  # Deploy with registry and hostname"
        echo "  $0 production delete                  # Delete production deployment"
        echo "  $0 development status                 # Show deployment status"
        exit 0
    fi
done

# Parse flags first and remove them from arguments
SKIP_TESTS=false
SKIP_PUSH=false
ARGS=()

for arg in "$@"; do
    case $arg in
        --skip-tests)
            SKIP_TESTS=true
            ;;
        --skip-push)
            SKIP_PUSH=true
            ;;
        *)
            ARGS+=("$arg")
            ;;
    esac
done

# Set positional arguments from remaining args
set -- "${ARGS[@]}"

# Default values
ENVIRONMENT="${1:-development}"
ACTION="${2:-apply}"
REGISTRY="${3:-}"
TAG="${4:-latest}"
HOSTNAME="${5:-}"

# Configuration
APP_NAME="health-coach"
if [ -n "$REGISTRY" ]; then
    IMAGE_TAG="${REGISTRY}/${APP_NAME}:${TAG}"
else
    IMAGE_TAG="${APP_NAME}:${TAG}"
fi
NAMESPACE="health-coach-${ENVIRONMENT}"
OVERLAY="$ENVIRONMENT"

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

# Function to validate parameters
validate_parameters() {
    print_status "Validating parameters..."
    
    # Validate environment
    if [[ ! "$ENVIRONMENT" =~ ^(development|staging|production)$ ]]; then
        print_warning "Environment '$ENVIRONMENT' is not a standard environment (development, staging, production)"
    fi
    
    # Validate action
    if [[ ! "$ACTION" =~ ^(apply|deploy|delete|destroy|status)$ ]]; then
        print_error "Invalid action '$ACTION'. Valid actions: apply, deploy, delete, destroy, status"
        exit 1
    fi
    
    # Validate registry format if provided
    if [ -n "$REGISTRY" ] && [[ ! "$REGISTRY" =~ ^[a-zA-Z0-9.-]+(\:[0-9]+)?$ ]]; then
        print_warning "Registry '$REGISTRY' might not be in the correct format"
    fi
    
    # Validate tag format
    if [[ ! "$TAG" =~ ^[a-zA-Z0-9._-]+$ ]]; then
        print_error "Invalid tag format '$TAG'. Tags should contain only alphanumeric characters, dots, underscores, and hyphens"
        exit 1
    fi
    
    print_success "Parameters validated"
}

# Function to run tests
run_tests() {
    print_status "Running tests..."
    
    if go test ./...; then
        print_success "All tests passed"
    else
        print_error "Tests failed"
        exit 1
    fi
}

# Function to build the Go application
build_app() {
    print_status "Building Go application..."
    
    # Set build variables
    export CGO_ENABLED=0
    export GOOS=linux
    export GOARCH=amd64
    
    # Build the application
    if go build -ldflags="-s -w" -o bin/health-coach cmd/server/main.go; then
        print_success "Application built successfully"
    else
        print_error "Failed to build application"
        exit 1
    fi
}

# Function to build Docker image
build_docker_image() {
    print_status "Building Docker image: $IMAGE_TAG"

    local container_engine="${CONTAINER_ENGINE:-podman}"
    
    if "$container_engine" build --file Dockerfile -t "$IMAGE_TAG" .; then
        print_success "Docker image built successfully"
        
        # Also tag as latest
        local latest_tag
        if [ -n "$REGISTRY" ]; then
            latest_tag="${REGISTRY}/${APP_NAME}:latest"
        else
            latest_tag="${APP_NAME}:latest"
        fi
        
        print_status "Tagging image as latest: $latest_tag"
        if "$container_engine" tag "$IMAGE_TAG" "$latest_tag"; then
            print_success "Image tagged as latest"
        else
            print_warning "Failed to tag image as latest"
        fi
    else
        print_error "Failed to build Docker image"
        exit 1
    fi
}

# Function to push Docker image
push_docker_image() {
    print_status "Pushing Docker image to registry..."
    
    if [ -z "$REGISTRY" ]; then
        print_warning "No registry specified, skipping push"
        return 0
    fi
    
    # Push the specific version tag
    if docker push "$IMAGE_TAG"; then
        print_success "Docker image pushed successfully: $IMAGE_TAG"
    else
        print_error "Failed to push Docker image: $IMAGE_TAG"
        exit 1
    fi
    
    # Also push the latest tag
    local latest_tag="${REGISTRY}/${APP_NAME}:latest"
    print_status "Pushing latest tag: $latest_tag"
    if docker push "$latest_tag"; then
        print_success "Latest tag pushed successfully: $latest_tag"
    else
        print_warning "Failed to push latest tag: $latest_tag"
    fi
}

# Function to create namespace if it doesn't exist
create_namespace() {
    print_status "Creating namespace if it doesn't exist..."
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        print_warning "Namespace $NAMESPACE already exists"
    else
        if kubectl create namespace "$NAMESPACE"; then
            print_success "Namespace $NAMESPACE created"
        else
            print_error "Failed to create namespace"
            exit 1
        fi
    fi
}

# Function to deploy to Kubernetes
deploy_to_kubernetes() {
    print_status "Deploying to Kubernetes..."
    
    # Check if overlay directory exists
    if [ ! -d "k8s/overlays/$OVERLAY" ]; then
        print_error "Overlay directory k8s/overlays/$OVERLAY does not exist"
        exit 1
    fi
    
    # Update the image tag in kustomization
    cd "k8s/overlays/$OVERLAY"
    
    # Update image tag
    kustomize edit set image "$APP_NAME=$IMAGE_TAG"
    
    # Update hostname in ingress if provided
    if [ -n "$HOSTNAME" ]; then
        print_status "Setting hostname to $HOSTNAME"
        # Replace HOSTNAME_PLACEHOLDER in ingress files (both base and overlay)
        find . -name "*.yaml" -type f -exec sed -i "s/HOSTNAME_PLACEHOLDER/$HOSTNAME/g" {} \;
        find ../../base -name "*.yaml" -type f -exec sed -i "s/HOSTNAME_PLACEHOLDER/$HOSTNAME/g" {} \;
    fi
    
    # Apply the configuration
    if kustomize build . | kubectl apply -f -; then
        print_success "Application deployed successfully"
    else
        print_error "Failed to deploy application"
        exit 1
    fi
    
    cd - > /dev/null
}

# Function to wait for rollout to complete
wait_for_rollout() {
    print_status "Waiting for deployment to complete..."
    
    # Get the actual deployment name with suffix
    DEPLOYMENT_NAME=$(kubectl get deployments -n "$NAMESPACE" -o name | grep health-coach | head -1 | cut -d'/' -f2)
    if [ -z "$DEPLOYMENT_NAME" ]; then
        print_error "No health-coach deployment found in namespace $NAMESPACE"
        exit 1
    fi
    
    print_status "Found deployment: $DEPLOYMENT_NAME"
    
    if kubectl rollout status deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" --timeout=300s; then
        print_success "Deployment completed successfully"
    else
        print_error "Deployment failed or timed out"
        exit 1
    fi
}

# Function to run health checks
run_health_checks() {
    print_status "Running health checks..."
    
    # Get the service URL
    local service_url
    if command -v minikube &> /dev/null && minikube status &> /dev/null; then
        # For minikube
        service_url=$(minikube service "$APP_NAME" -n "$NAMESPACE" --url)
    else
        # For other Kubernetes clusters, use port-forward
        print_status "Setting up port-forward for health check..."
        kubectl port-forward -n "$NAMESPACE" service/"$APP_NAME" 8080:80 &
        local port_forward_pid=$!
        sleep 5
        service_url="http://localhost:8080"
    fi
    
    # Check health endpoint
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$service_url/api/health" > /dev/null; then
            print_success "Health check passed"
            if [ ! -z "${port_forward_pid:-}" ]; then
                kill $port_forward_pid 2>/dev/null || true
            fi
            return 0
        fi
        
        print_status "Health check attempt $attempt/$max_attempts failed, retrying..."
        sleep 10
        ((attempt++))
    done
    
    print_error "Health check failed after $max_attempts attempts"
    if [ ! -z "${port_forward_pid:-}" ]; then
        kill $port_forward_pid 2>/dev/null || true
    fi
    exit 1
}

# Function to show deployment status
show_status() {
    print_status "Deployment status:"
    echo
    
    echo "Pods:"
    kubectl get pods -n "$NAMESPACE" -l app="$APP_NAME"
    echo
    
    echo "Services:"
    kubectl get services -n "$NAMESPACE"
    echo
    
    echo "Ingress:"
    kubectl get ingress -n "$NAMESPACE"
    echo
}

# Main deployment function
main() {
    print_status "Starting deployment of AI Health Coach application"
    print_status "Environment: $ENVIRONMENT"
    print_status "Action: $ACTION"
    print_status "Image: $IMAGE_TAG"
    print_status "Namespace: $NAMESPACE"
    print_status "Overlay: $OVERLAY"
    if [ -n "$HOSTNAME" ]; then
        print_status "Hostname: $HOSTNAME"
    fi
    echo
    
    # Run deployment steps based on action
    if [ "$ACTION" = "apply" ] || [ "$ACTION" = "deploy" ]; then
        validate_parameters
        if [ "$SKIP_TESTS" != "true" ]; then
            run_tests
        fi
        build_app
        build_docker_image
        if [ "$SKIP_PUSH" != "true" ] && [ -n "$REGISTRY" ]; then
            push_docker_image
        fi
        create_namespace
        deploy_to_kubernetes
        wait_for_rollout
        run_health_checks
        show_status
        
        print_success "Deployment completed successfully!"
        print_status "Application is now available at the configured ingress URL"
    elif [ "$ACTION" = "delete" ] || [ "$ACTION" = "destroy" ]; then
        validate_parameters
        print_status "Deleting deployment..."
        kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
        print_success "Deployment deleted successfully!"
    elif [ "$ACTION" = "status" ]; then
        validate_parameters
        show_status
    else
        print_error "Unknown action: $ACTION"
        print_error "Valid actions: apply, deploy, delete, destroy, status"
        exit 1
    fi
}

# Run main deployment
main
