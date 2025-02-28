# Work Order System Backend

This is the backend API for the Work Order System, built with Go, GoFiber, and PostgreSQL.

## Features

- **Authentication and Authorization**: JWT-based authentication with role-based access control.
- **Work Order Management**: Create, update, and view work orders.
- **Progress Tracking**: Track progress of work orders.
- **Reporting**: Generate reports on work orders and operator performance.

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher

## Installation

1. Clone the repository:

   ```
   git clone https://github.com/dawamr/work-order-system-go.git
   cd work-order-system-go/backend
   ```

2. Install dependencies:

   ```
   go mod download
   ```

3. Set up the database:

   - Create a PostgreSQL database
   - Update the `.env` file with your database credentials

4. Run the application:
   ```
   go run main.go
   ```

## Environment Variables

Create a `.env` file in the root directory with the following variables:

```
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=workorder

# JWT Configuration
JWT_SECRET=your-secret-key
TOKEN_EXPIRES_IN=24

# Server Configuration
PORT=8080
```

## Data Seeding

The application includes a data seeder to generate dummy data for testing and development purposes.

1. Make sure your database is running and configured correctly in the `.env` file.

2. Run the seeder:

   ```
   cd utils/seeder
   go run seeder.go
   ```

The seeder will generate:

- 1 Production Manager user (username: `manager`, password: `password`)
- 10 Operator users (usernames: `operator1` through `operator10`, password: `password`)
- 200 Work Orders with random statuses and progress entries
- All data is generated with dates between February 1, 2025, and February 28, 2025

For more details about the seeder, see the [Seeder README](utils/seeder/README.md).

## API Endpoints

### Authentication

- `POST /api/auth/login`: Login with username and password
- `POST /api/auth/register`: Register a new user

### Work Orders

- `GET /api/work-orders`: Get all work orders (Production Manager only)
- `POST /api/work-orders`: Create a new work order (Production Manager only)
- `GET /api/work-orders/:id`: Get a work order by ID
- `PUT /api/work-orders/:id`: Update a work order (Production Manager only)
- `GET /api/work-orders/assigned`: Get work orders assigned to the current operator (Operator only)
- `PUT /api/work-orders/:id/status`: Update a work order status (Operator only)

### Progress Tracking

- `POST /api/work-orders/:id/progress`: Add a progress entry to a work order
- `GET /api/work-orders/:id/progress`: Get progress entries for a work order
- `GET /api/work-orders/:id/history`: Get status history for a work order

### Reports

- `GET /api/reports/summary`: Get a summary of work orders by status (Production Manager only)
- `GET /api/reports/operators`: Get performance metrics for operators (Production Manager only)

## Project Structure

- `config/`: Configuration files and environment variable handling
- `controllers/`: Request handlers
- `database/`: Database connection and migration
- `middleware/`: Middleware functions for authentication and authorization
- `models/`: Data models
- `routes/`: API route definitions
- `utils/`: Utility functions and tools (including data seeder)

## License

This project is licensed under the MIT License.

## API Documentation

The API documentation is available through Swagger UI. After starting the server, you can access the documentation at:

```
http://localhost:8080/swagger/
```

### Features of the API Documentation:

- Interactive API documentation
- Try out API endpoints directly from the browser
- Authentication using JWT tokens
- Detailed request/response schemas
- Error responses documentation

### Generating Documentation

If you make changes to the API endpoints, you'll need to regenerate the Swagger documentation:

1. Install swag CLI tool:

   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

2. Generate documentation:

   ```bash
   cd backend
   swag init
   ```

3. Restart the server to see the changes.
