FROM debian:bookworm-slim AS build

WORKDIR /src
COPY --from=go-local go1.27.1.linux-amd64.tar.gz /tmp/go.tar.gz
RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates tar \
	&& tar -xzf /tmp/go.tar.gz -C /usr/local \
	&& rm -rf /var/lib/apt/lists/* /tmp/go.tar.gz

COPY go.mod go.sum ./
RUN /usr/local/go/bin/go mod download
COPY . .
RUN CGO_ENABLED=0 /usr/local/go/bin/go build -o /out/card2sheba ./cmd/api

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=build /out/card2sheba /app/card2sheba
COPY migrations /app/migrations
RUN mkdir -p /app/data

EXPOSE 8080
CMD ["/app/card2sheba"]