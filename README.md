# zchat

Generate shell commands from natural language using AI.

Supports **Ollama** (local, free) and **Anthropic Claude** (cloud, paid).

## Installation

```bash
git clone https://github.com/palaforcade/zchat
cd zchat
go build -o zchat
```

### Prerequisites

**Option 1: Ollama (Default)**
```bash
# Install Ollama
brew install ollama

# Pull a model
ollama pull qwen2.5-coder:7b
```

**Option 2: Anthropic**
```bash
export ZCHAT_PROVIDER=anthropic
export ANTHROPIC_API_KEY=sk-ant-api03-xxx
```

## Usage

```bash
./zchat <natural language query>
```

**Examples:**
```bash
./zchat list files in current directory
./zchat count lines in README.md
./zchat find all go files modified today
./zchat show disk usage sorted by size
```

## Configuration

**Default:** Uses Ollama with `qwen2.5-coder:7b` model.

**Environment Variables:**
```bash
export ZCHAT_PROVIDER=ollama           # or "anthropic"
export ANTHROPIC_API_KEY=sk-ant-xxx    # for Anthropic
export OLLAMA_URL=http://localhost:11434
```

**Config File:** `~/.config/zchat/config.yaml`
```yaml
provider: ollama
model: qwen2.5-coder:7b
ollama_url: http://localhost:11434
api_key: sk-ant-xxx  # for Anthropic
max_context_lines: 20
```

## Safety

Dangerous commands require explicit confirmation:
- `rm -rf /` - Recursive deletions
- `dd if=` - Disk operations
- `| sh` - Piped shell execution
- `diskutil` - Disk utilities

Configure via `dangerous_patterns` in config file.

## License

MIT
