# Model Prompt Guard Go - Multi-stage Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go-common package
COPY packages/go-common/ ./packages/go-common/

# Copy model service source
COPY apps/model-prompt-guard-go/ ./apps/model-prompt-guard-go/

# Build without go.work
RUN cd apps/model-prompt-guard-go && \
    GOWORK=off go mod download && \
    GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

COPY --from=builder /server /server

EXPOSE 8000

CMD ["/server"]
