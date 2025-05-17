# GOTH - Go Template for Hexagonal Architecture

GOTH is a clean, modern Go template for building robust backend applications based on hexagonal architecture (ports and adapters), inspired by [go-clean-template](https://github.com/evrone/go-clean-template).

## Features

- **Hexagonal Architecture**: Clean separation between domain logic and external adapters
- **Multi-platform Support**: Builds for amd64, arm, and arm64 architectures
- **Domain-Driven Structure**: Organized by business domains
- **Multiple Delivery Methods**: HTTP (Fiber) and AMQP RPC
- **Database Integration**: PostgreSQL with SQLC code generation
- **Authentication**: JWT-based authentication
- **Caching**: Redis integration
- **Dependency Injection**: Wire-based DI
- **Configuration**: Environment-based configuration
- **Logging**: Structured logging with zerolog
- **Documentation**: Swagger/OpenAPI integration
- **Testing**: Complete test suite with mocking
- **CI/CD**: GitHub Actions workflow
- **Docker Support**: Dockerized application with multi-stage builds
- **Hot Reload**: Development with Air for quick iterations

## Getting Started

### Prerequisites

- Go 1.24+
- Docker and Docker Compose
- Make

### Setup

1. Clone the repository:

```bash
git clone https://github.com/savioruz/goth.git
cd goth
```

2. Set up the environment variables:

```bash
cp .env.example .env
```

3. Run the application:

```bash
docker compose up -d
```

4. Run the migrations:

```bash
make migrate.up
```

### Dev

For development, you can use Air for hot reloading:

```bash
make dev
```

### Help

For help with the Makefile commands, run:

```bash
make help
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [go-clean-template](https://github.com/evrone/go-clean-template) - Inspiration for this project
- [Fiber](https://github.com/gofiber/fiber) - Fast HTTP framework
- [Wire](https://github.com/google/wire) - Dependency injection
- [SQLC](https://github.com/sqlc-dev/sqlc) - Type-safe SQL for Go
