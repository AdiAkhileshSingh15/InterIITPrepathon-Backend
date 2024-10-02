# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy && go build -o app .

# Run stage
FROM python:3.10-alpine

WORKDIR /app
COPY --from=builder /app/app .
COPY --from=builder /app/model.py .
COPY --from=builder /app/requirements.txt .
COPY --from=builder /app/.env .

# Install Python and pip using apk
RUN apk add --no-cache python3 py3-pip && \
    pip3 install --no-cache-dir -r requirements.txt

EXPOSE 8000
CMD ["./app"]
