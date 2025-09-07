# cmd/lb/Dockerfile
FROM golang:1.25.1-alpine AS build
WORKDIR /src

# copy module file from repo root (go.sum may not exist)
COPY go.mod ./
# optional: COPY go.sum ./ || true  -- cannot use || true in COPY, skip if not present

# copy full repo so builds can use internal packages
COPY . .

# fetch deps (creates go.sum inside image if needed)
RUN if [ -f go.mod ]; then go mod download; fi

# build to a non-conflicting path
ENV CGO_ENABLED=0
RUN GOOS=linux go build -trimpath -o /build/lb ./cmd/lb

# ---------- runtime ----------
FROM alpine:3.18
RUN apk add --no-cache ca-certificates

WORKDIR /app

# COPY the single binary (not a directory)
COPY --from=build /build/lb /app/lb

# ensure it's executable
RUN chmod +x /app/lb

EXPOSE 8080
ENTRYPOINT ["/app/lb"]
