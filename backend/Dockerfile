# Stage 1: build a static Go binary. This image is large (compiler, tools) and is thrown away.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

# Stage 2: copy only the binary into a tiny image.
FROM alpine:3.20
COPY --from=build /out/api /usr/local/bin/api
EXPOSE 8080
CMD ["api"]
