# backend/Dockerfile
FROM golang:1.25.1-alpine AS build

WORKDIR /src

# copy module files from repo root (build context is project root)
COPY go.mod ./
RUN if [ -f go.mod ]; then go mod download; fi

# copy full repo
COPY . .

# produce single binary (no name collision)
ENV CGO_ENABLED=0
RUN GOOS=linux go build -trimpath -o /build/backend ./backend

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /build/backend /app/backend
RUN chmod +x /app/backend
EXPOSE 8081
ENTRYPOINT ["/app/backend"]
