#!/bin/bash
set -e

# Check Go
if ! command -v go &> /dev/null; then
  echo "Go is required. Install from https://golang.org/dl"
  exit 1
fi

echo "Building B0uncer..."
go build -o b0uncer .

# Create config dir
mkdir -p ~/.b0uncer

# Save real bash path
REAL_BASH=$(which bash)
echo "{\"real_bash\": \"$REAL_BASH\"}" > ~/.b0uncer/config.json

# Copy policies if not already there
if [ ! -f ~/.b0uncer/policies.json ]; then
  cp policies.json ~/.b0uncer/policies.json
fi

# Install binary
if cp b0uncer /usr/local/bin/b0uncer 2>/dev/null; then
  echo "Installed to /usr/local/bin/b0uncer"
else
  sudo cp b0uncer /usr/local/bin/b0uncer
  echo "Installed to /usr/local/bin/b0uncer"
fi

echo ""
echo "Done. To use with Claude Code add this to .claude/settings.json:"
echo '{ "shell": "/usr/local/bin/b0uncer" }'
echo ""
echo "Or run: SHELL=/usr/local/bin/b0uncer claude"
echo "Dashboard: http://localhost:3456"
