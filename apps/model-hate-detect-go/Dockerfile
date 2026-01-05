# Model Hate Detect Go - Multi-stage Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY packages/go-common/ ./packages/go-common/
COPY apps/model-hate-detect-go/ ./apps/model-hate-detect-go/

RUN cd apps/model-hate-detect-go && \
    GOWORK=off go mod download && \
    GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /server /server
EXPOSE 8000
CMD ["/server"]
