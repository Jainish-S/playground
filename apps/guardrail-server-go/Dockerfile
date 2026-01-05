# Guardrail Server Go - Multi-stage Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go-common package first
COPY packages/go-common/ ./packages/go-common/

# Copy guardrail-server-go source
COPY apps/guardrail-server-go/ ./apps/guardrail-server-go/

# Build without go.work (use go.mod replace directive instead)
RUN cd apps/guardrail-server-go && \
    GOWORK=off go mod download && \
    GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /

COPY --from=builder /server /server

EXPOSE 8000

CMD ["/server"]
