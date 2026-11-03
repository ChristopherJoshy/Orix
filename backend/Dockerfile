# Staged Go image. Build/run only after code completion and environment setup.
FROM golang:1.27.3-trixie AS build
WORKDIR /source
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY app/*.go app/*.html app/*.js app/*.css ./app/
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /orix .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /orix /orix
USER 65532:65532
WORKDIR /app
ENV LISTEN_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/orix"]
