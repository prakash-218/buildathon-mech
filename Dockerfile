# Build a static Go binary, ship it in an empty image.
FROM golang:1.24.11-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /mechbot .

FROM scratch
COPY --from=build /mechbot /mechbot
ENV MECH_FOCUS=1 MECH_FFA=1 MECH_FFA_THREAT=0 MECH_NOOK=1 MECH_CRATE_HUNT=150
USER 65534:65534
ENTRYPOINT ["/mechbot"]
