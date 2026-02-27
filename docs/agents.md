# Agents & Providers (v0)

Krellin supports multiple LLM providers via a provider registry stored at:

- `$KRELLIN_HOME/providers.json` (defaults to `~/.krellin/providers.json`)

## Supported provider types

- `openai`
- `anthropic`
- `grok`
- `llama` (self-hosted, OpenAI-compatible)

## CLI Workflow

List providers:

```
krellin providers list
```

Add a provider:

```
krellin providers add --name myopenai --type openai --model gpt-4o-mini --api-key-env OPENAI_API_KEY
krellin providers add --name myanthropic --type anthropic --model claude-3-5-sonnet --api-key-env ANTHROPIC_API_KEY
krellin providers add --name mygrok --type grok --model grok-2 --api-key-env GROK_API_KEY --base-url https://api.x.ai/v1
krellin providers add --name myllama --type llama --model llama3 --api-key-env LLAMA_API_KEY --base-url http://localhost:8000/v1
```

## Notes

- API keys are read from environment variables to avoid storing secrets on disk.
- Base URLs are optional for hosted providers and required for self-hosted LLaMA.
