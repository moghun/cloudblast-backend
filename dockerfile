# Use an official Golang runtime as the base image
FROM golang:1.19

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod .
COPY go.sum .

# Download dependencies
RUN go mod download

# Copy the rest of the application code to the working directory
COPY . .

# Build the application
RUN go build -o main ./cmd/main.go

# Expose the port the application runs on
EXPOSE 8080

# Set AWS credentials and region for AWS SDK
#ENV AWS_ACCESS_KEY_ID=
#ENV AWS_SECRET_ACCESS_KEY=

# Download and run RabbitMQ and Redis instances
RUN apt-get update && apt-get install -y rabbitmq-server redis-server

# Command to start RabbitMQ, Redis, and the application
CMD service rabbitmq-server start && service redis-server start && ./main
