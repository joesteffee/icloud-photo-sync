FROM golang:1.18-alpine AS builder

RUN apk add ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o icloud-photo-sync .

FROM alpine:3
RUN apk add ca-certificates
COPY --from=builder /app/icloud-photo-sync /
WORKDIR /images
ENTRYPOINT ["/icloud-photo-sync"]

