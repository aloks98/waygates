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

**Error Responses**

- **500 Internal Server Error**: If the database query to count users fails.
  ```json
  {
    "success": false,
    "error": {
      "code": "INTERNAL_ERROR",
      "message": "Failed to check user status"
    }
  }
  ```

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

**Error Responses**

- **400 Bad Request**: If the request body is invalid or missing required fields.
  ```json
  {
    "success": false,
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "Name, username, email, and password are required"
    }
  }
  ```
- **409 Conflict**: If a user with the same username or email already exists.
  ```json
  {
    "success": false,
    "error": {
      "code": "CONFLICT",
      "message": "User with this email or username already exists"
    }
  }
  ```
- **500 Internal Server Error**: For any other server-side errors.

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

**Error Responses**

- **400 Bad Request**: If the request body is invalid.
- **401 Unauthorized**: If the credentials are a invalid.
  ```json
  {
    "success": false,
    "error": {
      "code": "UNAUTHORIZED",
      "message": "Invalid credentials"
    }
  }
  ```
- **500 Internal Server Error**: For any other server-side errors.

### 4. Refresh Token

Exchanges a valid refresh token for a new access token and refresh token pair.

- **Endpoint**: `POST /api/auth/refresh`
- **Access**: Public

**Request Body**

```json
{
  "refresh_token": "ey..."
}
```

**Success Response (200 OK)**

```json
{
  "success": true,
  "message": "Token refreshed successfully",
  "data": {
    "access_token": "ey...",
    "refresh_token": "ey..."
  }
}
```

**Error Responses**

- **400 Bad Request**: If the refresh token is missing.
  ```json
  {
    "success": false,
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "Refresh token is required"
    }
  }
  ```
- **401 Unauthorized**: If the refresh token is invalid or expired.
  ```json
  {
    "success": false,
    "error": {
      "code": "UNAUTHORIZED",
      "message": "Invalid or expired refresh token"
    }
  }
  ```

### 5. Logout

Revokes the current access token and optionally the refresh token.

- **Endpoint**: `POST /api/auth/logout`
- **Access**: Authenticated (requires Bearer token)

**Request Headers**

```
Authorization: Bearer <access_token>
```

**Request Body (Optional)**

```json
{
  "refresh_token": "ey..."
}
```

**Success Response (200 OK)**

```json
{
  "success": true,
  "message": "Logged out successfully",
  "data": null
}
```

**Error Responses**

- **400 Bad Request**: If no token is provided in the Authorization header.

**Notes**:
- The access token is always revoked.
- If a refresh token is provided in the body, it will also be revoked.
- For complete logout, include both the access token (in header) and refresh token (in body).

### 6. Get Current User

Returns information about the currently authenticated user, including their role and permissions.

- **Endpoint**: `GET /api/auth/me`
- **Access**: Authenticated (requires Bearer token)

**Request Headers**

```
Authorization: Bearer <access_token>
```

**Success Response (200 OK)**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "Alice",
    "username": "alice",
    "email": "alice@example.com",
    "role": "admin",
    "permissions": [
      "proxy:read",
      "proxy:create",
      "proxy:update",
      "proxy:delete",
      "user:read",
      "user:manage"
    ]
  }
}
```

**Response Fields**

- `role`: The user's assigned role (`"admin"`, `"operator"`, or `null` if no role assigned)
- `permissions`: Array of permission strings the user has

**Error Responses**

- **401 Unauthorized**: If the token is missing or invalid.
- **404 Not Found**: If the user no longer exists.

## Authentication Flow

1.  **Check Status**: Before registration, the UI can call `GET /api/status` to see if `user_setup_complete` is `false`. If so, it can show the registration page to create the first admin user.
2.  **Register**: The first user (or any subsequent user) registers via `POST /api/auth/register`.
3.  **Login**: The user logs in via `POST /api/auth/login` to get an `access_token` and `refresh_token`.
4.  **Access Protected Endpoints**: To access protected endpoints (like the proxy management endpoints), include the `access_token` in the `Authorization` header.

    ```
    Authorization: Bearer <access_token>
    ```

5.  **Token Expiration**: The access token is short-lived (e.g., 15 minutes). When it expires, the API will return a `401 Unauthorized` error. The client should use the `refresh_token` with `POST /api/auth/refresh` to get a new token pair.

6.  **Token Refresh**: Call `POST /api/auth/refresh` with the refresh token to obtain new access and refresh tokens. This implements token rotation - each refresh invalidates the old refresh token and issues a new one.

7.  **Logout**: Call `POST /api/auth/logout` with both the access token (in Authorization header) and optionally the refresh token (in body) to fully revoke the session.

8.  **Get User Info**: Call `GET /api/auth/me` to retrieve the current user's profile, role, and permissions.

## Default User Creation

On startup, the backend can create a default user if the following environment variables are set and no other users exist in the database:

- `DEFAULT_USER_NAME`
- `DEFAULT_USER_USERNAME`
- `DEFAULT_USER_EMAIL`
- `DEFAULT_USER_PASSWORD`

If these variables are not set, no default user will be created, and the first user must be registered through the API.
