FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/engo ./cmd/engo

FROM alpine:3.22
RUN addgroup -S engo && adduser -S -G engo engo
WORKDIR /app
COPY --from=build /out/engo /usr/local/bin/engo
COPY scripts ./scripts
USER engo
ENTRYPOINT ["/usr/local/bin/engo"]
