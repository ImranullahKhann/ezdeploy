FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/backend ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/worker ./cmd/worker

FROM alpine:3.20

# Install git and docker CLI
RUN apk add --no-cache git docker-cli

WORKDIR /app
COPY --from=build /out/backend /app/backend
COPY --from=build /out/worker /app/worker
COPY --from=build /src/migrations /app/migrations

EXPOSE 8080
CMD ["/app/backend"]
