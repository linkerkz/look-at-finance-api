#!/bin/bash
#
# Git Hooks Setup Script
# Configures git to use the .githooks directory
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}=== Git Hooks Setup ===${NC}"
echo ""

# Navigate to project root
cd "$PROJECT_ROOT"

# ============================================================
# 1. Configure git to use .githooks directory
# ============================================================
echo -e "${BLUE}[1/3] Configuring git hooks path...${NC}"

git config core.hooksPath .githooks
echo -e "${GREEN}DONE${NC} - Git will now use .githooks/ directory"
echo ""

# ============================================================
# 2. Make hooks executable
# ============================================================
echo -e "${BLUE}[2/3] Making hooks executable...${NC}"

chmod +x .githooks/pre-commit 2>/dev/null || true
chmod +x .githooks/pre-push 2>/dev/null || true
echo -e "${GREEN}DONE${NC}"
echo ""

# ============================================================
# 3. Check and install optional tools
# ============================================================
echo -e "${BLUE}[3/3] Checking required tools...${NC}"
echo ""

MISSING_TOOLS=0

# gofmt (usually comes with Go)
if command -v gofmt &> /dev/null; then
    echo -e "  ${GREEN}[OK]${NC} gofmt"
else
    echo -e "  ${RED}[MISSING]${NC} gofmt - Install Go"
    MISSING_TOOLS=1
fi

# goimports
if command -v goimports &> /dev/null; then
    echo -e "  ${GREEN}[OK]${NC} goimports"
else
    echo -e "  ${YELLOW}[MISSING]${NC} goimports"
    echo -e "       Install: go install golang.org/x/tools/cmd/goimports@latest"
    MISSING_TOOLS=1
fi

# golangci-lint
if command -v golangci-lint &> /dev/null; then
    echo -e "  ${GREEN}[OK]${NC} golangci-lint"
else
    echo -e "  ${YELLOW}[MISSING]${NC} golangci-lint"
    echo -e "       Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
    MISSING_TOOLS=1
fi

# govulncheck
if command -v govulncheck &> /dev/null; then
    echo -e "  ${GREEN}[OK]${NC} govulncheck"
else
    echo -e "  ${YELLOW}[MISSING]${NC} govulncheck (optional)"
    echo -e "       Install: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

echo ""

# ============================================================
# Summary
# ============================================================
echo -e "${BLUE}=== Setup Complete ===${NC}"
echo ""
echo "Git hooks are now enabled:"
echo "  - pre-commit: gofmt, goimports, lint, unit tests"
echo "  - pre-push: full tests, static analysis, security scan"
echo ""

if [ $MISSING_TOOLS -ne 0 ]; then
    echo -e "${YELLOW}Some optional tools are missing. Install them for full functionality.${NC}"
    echo ""
fi

echo -e "To install all recommended tools at once:"
echo -e "${YELLOW}go install golang.org/x/tools/cmd/goimports@latest && \\"
echo -e "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \\"
echo -e "go install golang.org/x/vuln/cmd/govulncheck@latest${NC}"
echo ""
echo -e "Bypass hooks when needed:"
echo -e "  git commit --no-verify"
echo -e "  git push --no-verify"
