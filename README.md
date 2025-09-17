# Gin Rate Limiter

A versatile, and pluggable rate-limiting middleware for the [Gin](https://github.com/gin-gonic/gin) web framework.

This package provides middleware for several popular rate-limiting algorithms and is designed for both single-instance and distributed applications through a pluggable storage interface.

## Features 
-   **Multiple Algorithms**: Includes implementations for Token Bucket, Leaky Bucket, Fixed Window, Sliding Window Log, and Sliding Window Counter.
-   **Pluggable Storage**: Comes with a built-in in-memory store for development and a Redis store for distributed, production environments.
