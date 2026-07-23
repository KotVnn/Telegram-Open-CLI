# Basic TOC Example

This example demonstrates basic TOC usage.

## Setup

1. Copy the example config:
   ```bash
   cp ../../configs/example.toml ~/.toc/config.toml
   ```

2. Edit the config with your Telegram bot token

3. Start the bot:
   ```bash
   toc start
   ```

4. Open Telegram and find your bot

5. Send `/start` to begin

## Commands

| Command | Description |
|---------|-------------|
| `/start` | Start conversation |
| `/help` | Show help |
| `/new myproject` | Create new session |
| `/sessions` | List sessions |
| `/status` | Show current status |

## Usage

```
You: /new my-api
Bot: Created session "my-api" using OpenCode

You: Fix the bug in user.go
Bot: Processing...
Bot: Found the issue in user.go:42
Bot: Applied changes. Want me to run tests?

You: Yes
Bot: Running tests...
Bot: All tests passing!
```
