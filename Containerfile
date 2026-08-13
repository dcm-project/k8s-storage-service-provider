# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:1.25.5 AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

USER root
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -o k8s-storage-service-provider ./cmd/k8s-storage-service-provider

# Runtime stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

WORKDIR /app

COPY --from=builder /app/k8s-storage-service-provider .

EXPOSE 8080

ENTRYPOINT ["./k8s-storage-service-provider"]
