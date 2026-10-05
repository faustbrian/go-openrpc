# API compatibility baseline

`v1.export` is the retained published-v1 baseline. `v2.export` is module export
data for the next major source pending its first v2 tag, generated with the
pinned `apidiff` version:

```sh
go run golang.org/x/exp/cmd/apidiff@v0.0.0-20260709172345-9ea1abe57597 \
  -m -w api/v2.export github.com/faustbrian/go-openrpc/v2
```

`golib api check` rejects incompatible exported API changes relative to this
baseline.
