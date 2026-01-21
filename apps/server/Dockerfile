# syntax=docker/dockerfile:1

# -------- builder --------
FROM golang:1.24 AS build
ARG SERVER_DIR=apps/server

WORKDIR /src

# Prime module cache for faster rebuilds
COPY ${SERVER_DIR}/go.mod ${SERVER_DIR}/go.sum ./
RUN go mod download

# Bring in source last for cacheability
COPY ${SERVER_DIR}/ ./

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/server .

# -------- runtime --------
FROM gcr.io/distroless/static-debian12
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
COPY --from=build /bin/server /server
ENTRYPOINT ["/server"]
