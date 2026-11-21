# Mech Royale bot: combined brain (branch `exp/hybrid`)

- **Practice (house squad):** plays **giddygiddyboy-mech-royale** by giddygiddyboy
  (github.com/justnau/giddygiddyboy-mech-royale, commit `d5b5657`), package `mech/`, unmodified,
  under its MIT License (see `LICENSE`, which keeps the original copyright notice).
- **Free-for-all:** once our own brain decides the match is a free-for-all (3 or more mechs died
  that we didn't kill), control switches to **our own bot** (package `ours/`, from this
  repository's `main` branch) with its free-for-all profile. Both brains see every observation
  from the start.

The glue is `cmd/bot/main.go`. Our standalone bot is on `main`.

```sh
docker build --platform linux/amd64 -t bot .
```
