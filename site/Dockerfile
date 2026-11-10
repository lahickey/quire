# The documentation site's image: one static binary with the baked site embedded.
# Built from the repo root, because the site is part of the quire module.
# Bake first (`just site bake`): the build reads site/.quire.
# quire_nobake: the bake has already run, so the server carries none of it.
# -overlay: adds the ship.go the bake wrote, which embeds the bundle.

FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN cd site && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags quire_nobake -overlay .quire/ship.json -ldflags="-s -w" -o /server .

FROM scratch
COPY --from=build /server /server
EXPOSE 8080
USER 10001:10001
CMD ["/server", "--addr", ":8080"]
