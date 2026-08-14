# oeh-cli

**Open Engineering Hub** — Official CLI tool for verifying and submitting engineering tasks.

Inspired by [Boot.dev CLI](https://github.com/bootdotdev/bootdev) and [freeCodeCamp workspaces](https://www.freecodecamp.org/), but built for real engineering verification — local checks, evidence collection, and authoritative remote validation.

---

## Installation

### Requirements
- Go 1.22+

### Install
```bash
go install github.com/open-engineering-hub/oeh-cli@latest
```

Verify:
```bash
oeh --version
```

---

## Quick Start

### 1. Get your token
Go to **Settings → CLI & Workspace** on the OEH platform and click **Generate & Copy**.

### 2. Login
```bash
oeh login --token YOUR_TOKEN
```

### 3. Check your environment
```bash
oeh doctor
```

### 4. Start a task
```bash
oeh task start ie-ch03-lab-001
```

### 5. Do the work, then verify
```bash
oeh verify
```

### 6. Submit
```bash
oeh submit
```

---

## Commands

| Command | Description |
|---------|-------------|
| `oeh login --token TOKEN` | Authenticate with your OEH account |
| `oeh logout` | Remove stored credentials |
| `oeh doctor` | Check your engineering environment |
| `oeh task start ID` | Initialize workspace for a task |
| `oeh task status` | Show current task and verification steps |
| `oeh verify` | Run local verification checks |
| `oeh submit` | Submit to platform for authoritative verification |
| `oeh submission status` | Check submission status |

---

## How It Works

```
Platform (Task ID)
        ↓
oeh task start ie-ch03-lab-001
        ↓
.oeh/ workspace created
        ↓
Do the engineering work...
        ↓
oeh verify   ← runs local checks
        ↓
oeh submit   ← sends evidence to OEH
        ↓
Platform runs authoritative verification
        ↓
XP + Progress update
```

### Local Verification
The CLI runs checks locally before submission:
- **HTTP checks** — probes running servers
- **Process checks** — detects running tools
- **File checks** — verifies required files exist
- **Command checks** — validates command output

### Workspace Directory
```
your-project/
├── .oeh/
│   ├── config.json      ← CLI config
│   ├── task.json        ← Task spec from platform
│   ├── state.json       ← Workspace state
│   └── evidence/        ← Collected evidence
└── ...your code...
```

---

## Task ID Format

```
ie-ch03-lab-001
│  │    │   │
│  │    │   └── Task number
│  │    └────── Type: lab | prj | chg | inc
│  └─────────── Chapter
└────────────── Course: ie | sys | ai | be | sec
```

---

## Workspace Modes

### Local Machine (Recommended)
```bash
git clone ...
cd project
oeh login --token TOKEN
oeh task start ie-ch03-lab-001
```

### GitHub Codespaces
Click "Open in Codespace" on the task page. The CLI comes pre-installed.

### VS Code Dev Container
Clone the repo, open in VS Code, reopen in container. CLI pre-installed.

---

## Config

Config is stored at `~/.oeh/config.json`:
```json
{
  "auth": {
    "token": "your-token",
    "platform_url": "https://open-engineering-hub.dev"
  }
}
```

Override the platform URL:
```bash
oeh login --token TOKEN
# config saves to ~/.oeh/config.json
```

---

## Development

```bash
git clone https://github.com/open-engineering-hub/oeh-cli
cd oeh-cli
go build -o oeh .
./oeh doctor
```

---

## License

MIT
