# Authentication API

The backend provides a JWT-based authentication system. All API endpoints, except for the health, status, and auth endpoints themselves, are protected and require a valid access token.

## Endpoints

### 1. Get Application Status

Provides the status of the Caddy server and whether the initial user setup has been completed.

- **Endpoint**: `GET /api/status`
- **Access**: Public

**Success Response (200 OK)**

```json
{
  "success": true,
  "message": "Status retrieved successfully",
  "data": {
    "caddy_status": "healthy",
    "user_setup_complete": true
  }
}
```

- `caddy_status`: "healthy" or "unhealthy"
- `user_setup_complete`: `true` if at least one user exists in the database, `false` otherwise.

### 2. Register a New User

Creates a new user account.

- **Endpoint**: `POST /api/auth/register`
- **Access**: Public

**Request Body**

```json
{
  "name": "Alice",
  "username": "alice",
  "email": "alice@example.com",
  "password": "securepassword123"
}
```

**Success Response (201 Created)**

```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "id": 1,
    "name": "Alice",
    "username": "alice",
    "email": "alice@example.com",
    "created_at": "2025-11-15T12:00:00Z",
    "updated_at": "2025-11-15T12:00:00Z"
  }
}
```

### 3. User Login

Authenticates a user and returns access and refresh tokens.

- **Endpoint**: `POST /api/auth/login`
- **Access**: Public

**Request Body**

You can log in with either your `username` or `email`.

```json
{
  "identifier": "alice", // or "alice@example.com"
  "password": "securepassword123"
}
```

**Success Response (200 OK)**

```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "access_token": "ey...",
    "refresh_token": "ey..."
  }
}
```

## Authentication Flow

1.  **Check Status**: Before registration, the UI can call `GET /api/status` to see if `user_setup_complete` is `false`. If so, it can show the registration page to create the first admin user.
2.  **Register**: The first user (or any subsequent user) registers via `POST /api/auth/register`.
3.  **Login**: The user logs in via `POST /api/auth/login` to get an `access_token` and `refresh_token`.
4.  **Access Protected Endpoints**: To access protected endpoints (like the proxy management endpoints), include the `access_token` in the `Authorization` header.

    ```
    Authorization: Bearer <access_token>
    ```

5.  **Token Expiration**: The access token is short-lived (e.g., 15 minutes). When it expires, the API will return a `401 Unauthorized` error. The client should then use the `refresh_token` to get a new access token (refresh endpoint not yet implemented).

## Default User Creation

On startup, the backend can create a default user if the following environment variables are set and no other users exist in the database:

- `DEFAULT_USER_NAME`
- `DEFAULT_USER_USERNAME`
- `DEFAULT_USER_EMAIL`
- `DEFAULT_USER_PASSWORD`

If these variables are not set, no default user will be created, and the first user must be registered through the API.
