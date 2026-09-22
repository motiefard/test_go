FROM golang:1.27-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/card2sheba ./cmd/api

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=build /out/card2sheba /app/card2sheba
COPY migrations /app/migrations
RUN mkdir -p /app/data

EXPOSE 8080
CMD ["/app/card2sheba"]