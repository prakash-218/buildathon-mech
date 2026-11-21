# Mech Royale bot (branch: giddygiddyboy strategy)

This branch runs **giddygiddyboy-mech-royale** by giddygiddyboy (github.com/justnau/giddygiddyboy-mech-royale),
commit `d5b5657`, under its MIT License (see `LICENSE`, which keeps the original copyright notice).

Our own bot is on the `main` branch of this repository.

```sh
docker build --platform linux/amd64 -t bot .
```

Change on this branch: the random seed is offset by one (`mech/brain.go`, `NewBrain`), nothing else.
