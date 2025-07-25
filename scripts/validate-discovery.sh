#!/bin/bash

# Compass MCP Discovery Validation Script
# This script validates that all discovery endpoints are working correctly
# and helps maintain accurate documentation in CLAUDE.md

set -e

COMPASS_BIN="./bin/compass"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 Compass MCP Discovery Validation"
echo "===================================="

# Check if binary exists
if [ ! -f "$COMPASS_BIN" ]; then
    echo -e "${RED}❌ Compass binary not found at $COMPASS_BIN${NC}"
    echo "Run: go build -o bin/compass cmd/compass/main.go"
    exit 1
fi

echo -e "${GREEN}✅ Found Compass binary${NC}"

# Test tools/list endpoint
echo ""
echo "🛠️  Testing tools/list endpoint..."
TOOLS_COUNT=$(echo '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}' | $COMPASS_BIN 2>/dev/null | jq -r '.result.tools | length' 2>/dev/null || echo "ERROR")

if [ "$TOOLS_COUNT" = "ERROR" ]; then
    echo -e "${RED}❌ tools/list endpoint failed${NC}"
    exit 1
else
    echo -e "${GREEN}✅ tools/list: $TOOLS_COUNT tools discovered${NC}"
fi

# Test resources/list endpoint  
echo ""
echo "📊 Testing resources/list endpoint..."
RESOURCES_COUNT=$(echo '{"jsonrpc": "2.0", "id": 1, "method": "resources/list"}' | $COMPASS_BIN 2>/dev/null | jq -r '.result.resources | length' 2>/dev/null || echo "ERROR")

if [ "$RESOURCES_COUNT" = "ERROR" ]; then
    echo -e "${RED}❌ resources/list endpoint failed${NC}"
    exit 1
else
    echo -e "${GREEN}✅ resources/list: $RESOURCES_COUNT resources discovered${NC}"
fi

# Test prompts/list endpoint
echo ""
echo "💡 Testing prompts/list endpoint..."
PROMPTS_COUNT=$(echo '{"jsonrpc": "2.0", "id": 1, "method": "prompts/list"}' | $COMPASS_BIN 2>/dev/null | jq -r '.result.prompts | length' 2>/dev/null || echo "ERROR")

if [ "$PROMPTS_COUNT" = "ERROR" ]; then
    echo -e "${RED}❌ prompts/list endpoint failed${NC}"
    exit 1
else
    echo -e "${GREEN}✅ prompts/list: $PROMPTS_COUNT prompts discovered${NC}"
fi

# Test sample resource reading
echo ""
echo "📖 Testing resource reading..."
RESOURCE_TEST=$(echo '{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": {"uri": "compass://processes"}}' | $COMPASS_BIN 2>/dev/null | jq -r '.result.contents[0].text' 2>/dev/null || echo "ERROR")

if [[ "$RESOURCE_TEST" == *"Process List"* ]]; then
    echo -e "${GREEN}✅ Resource reading works correctly${NC}"
elif [ "$RESOURCE_TEST" = "ERROR" ]; then
    echo -e "${RED}❌ Resource reading failed${NC}"
    exit 1
else
    echo -e "${YELLOW}⚠️  Resource reading returned unexpected result${NC}"
fi

# Check for process-specific tools
echo ""
echo "🔄 Validating process management tools..."
PROCESS_TOOLS=$(echo '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}' | $COMPASS_BIN 2>/dev/null | jq -r '.result.tools[] | select(.name | startswith("compass_process")) | .name' 2>/dev/null | wc -l || echo "0")

if [ "$PROCESS_TOOLS" -ge "10" ]; then
    echo -e "${GREEN}✅ Process management tools: $PROCESS_TOOLS tools found${NC}"
else
    echo -e "${YELLOW}⚠️  Expected 10+ process tools, found: $PROCESS_TOOLS${NC}"
fi

# Summary and CLAUDE.md update check
echo ""
echo "📋 Discovery Summary:"
echo "===================="
echo "Tools:     $TOOLS_COUNT"
echo "Resources: $RESOURCES_COUNT" 
echo "Prompts:   $PROMPTS_COUNT"
echo ""

# Check if CLAUDE.md needs updating
CLAUDE_MD="CLAUDE.md"
if [ -f "$CLAUDE_MD" ]; then
    if grep -q "tools/list:.*$TOOLS_COUNT tools" "$CLAUDE_MD" && \
       grep -q "resources/list:.*$RESOURCES_COUNT resources" "$CLAUDE_MD" && \
       grep -q "prompts/list:.*$PROMPTS_COUNT prompts" "$CLAUDE_MD"; then
        echo -e "${GREEN}✅ CLAUDE.md inventory appears up-to-date${NC}"
    else
        echo -e "${YELLOW}⚠️  CLAUDE.md inventory may need updating${NC}"
        echo "Update the 'MCP Capabilities Inventory' section with:"
        echo "  tools/list:     $TOOLS_COUNT tools"
        echo "  resources/list: $RESOURCES_COUNT resources"
        echo "  prompts/list:   $PROMPTS_COUNT prompts"
    fi
else
    echo -e "${YELLOW}⚠️  CLAUDE.md not found${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Discovery validation complete!${NC}"