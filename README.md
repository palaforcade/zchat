# zchat

Generate shell commands from natural language using Claude AI.

## Installation

1. Download the binary for your platform from releases
2. Move to your PATH:
   ```bash
   mv zchat-* ~/.local/bin/zchat
   chmod +x ~/.local/bin/zchat
   ```
3. Set your Anthropic API key:
   ```bash
   export ANTHROPIC_API_KEY=sk-ant-api03-xxx
   ```

## Usage

```bash
zchat <natural language query>
```

### Examples

```bash
# File operations
zchat list the number of lines in data.csv
zchat find all python files modified in the last week

# Text processing
zchat search for "error" in all log files
zchat count unique words in README.md

# System info
zchat show disk usage sorted by size
zchat list running processes using more than 1GB memory
```

## Configuration

### Environment Variable (Recommended)
```bash
export ANTHROPIC_API_KEY=sk-ant-api03-xxx
```

### Config File (Optional)
Create `~/.config/zchat/config.yaml`:
```yaml
api_key: sk-ant-api03-xxx
model: claude-sonnet-4-5-20250929
max_context_lines: 20
```

## Safety Features

zchat detects and warns about potentially dangerous commands:
- Recursive deletions (`rm -rf`)
- Disk operations (`dd`, `mkfs`)
- Commands that download and execute code
- Fork bombs and system disruption

Dangerous commands require explicit confirmation before execution.

## Building from Source

```bash
git clone https://github.com/palaforcade/zchat
cd zchat
go build -o zchat
```

## License

MIT
