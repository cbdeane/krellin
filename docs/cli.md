# CLI (v0)

## krellin

Starts the TUI client and connects to the local daemon over the unix socket.

Flags:
- `-repo` repo root (defaults to cwd)
- `-sock` unix socket (defaults to `/tmp/krellin.sock`)

### Provider management

```
krellin providers list
krellin providers add --name myopenai --type openai --model gpt-4o-mini --api-key-env OPENAI_API_KEY
krellin providers add --name myanthropic --type anthropic --model claude-3-5-sonnet --api-key-env ANTHROPIC_API_KEY
krellin providers add --name mygrok --type grok --model grok-2 --api-key-env GROK_API_KEY --base-url https://api.x.ai/v1
krellin providers add --name myllama --type llama --model llama3 --api-key-env LLAMA_API_KEY --base-url http://localhost:8000/v1
```

## krellind

Starts the daemon and listens on the unix socket.

Flags:
- `-sock` unix socket (defaults to `/tmp/krellin.sock`)
