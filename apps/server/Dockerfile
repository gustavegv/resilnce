# syntax=docker/dockerfile:1

FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -v -x -ldflags="-s -w" -o /bin/server ./main.go


FROM gcr.io/distroless/static-debian12
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
COPY --from=build /bin/server /server
ENTRYPOINT ["/server"]
