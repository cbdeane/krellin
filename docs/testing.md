# Testing (v0)

## Unit

```
GOCACHE=/tmp/krellin-gocache go test ./...
```

## Integration/E2E

Some tests require sockets and Docker. Enable with:

```
KRELLIN_E2E=1 GOCACHE=/tmp/krellin-gocache go test ./internal/integration -v
```

If the environment disallows unix sockets, tests will skip.
