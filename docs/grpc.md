# gRPC API

The service exposes a gRPC API on port `50051`.

The gRPC API is defined in:

```text
proto/api/user/v1/user_service.proto
```

Generated Go files are committed under:

```text
internal/gen/api/user/v1/
```

## Service

```text
api.user.v1.UserService
```

Available RPC methods:

```text
GetUser
ListUsers
```

## Authentication

gRPC requests use the same JWT access token as the REST API.

Pass the token through gRPC metadata:

```text
authorization: Bearer <access-token>
```

In `local` environment, gRPC reflection is enabled. This allows tools such as `grpcurl` to inspect and call the service
without manually passing `.proto` files.

In non-local environments, reflection is disabled. Use `-import-path` and `-proto` with `grpcurl`.

## Start the service

Start the local development stack:

```bash
docker compose up --build
```

The gRPC server will be available at:

```text
localhost:50051
```

## Install grpcurl

Using Go:

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

Make sure your Go bin directory is available in `PATH`.

On Windows PowerShell, you can check it with:

```powershell
go env GOPATH
```

The binary is usually located in:

```text
%USERPROFILE%\go\bin
```

## List available services

With reflection enabled:

```bash
grpcurl -plaintext localhost:50051 list
```

Expected services include:

```text
api.user.v1.UserService
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
```

## Describe the user service

```bash
grpcurl -plaintext localhost:50051 describe api.user.v1.UserService
```

Expected output includes:

```text
service UserService {
  rpc GetUser ( .api.user.v1.GetUserRequest ) returns ( .api.user.v1.GetUserResponse );
  rpc ListUsers ( .api.user.v1.ListUsersRequest ) returns ( .api.user.v1.ListUsersResponse );
}
```

## Register a test user

Use the REST API to register a user and receive JWT tokens:

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Alex",
    "email": "alex@example.com",
    "age": 25,
    "password": "password123"
  }'
```

Example response:

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "tokenType": "Bearer",
  "refreshToken": "refresh-token-value",
  "user": {
    "id": 1,
    "name": "Alex",
    "email": "alex@example.com",
    "age": 25,
    "role": "USER",
    "createdAt": "2026-08-24T10:00:00Z",
    "updatedAt": "2026-08-24T10:00:00Z"
  }
}
```

Save the `accessToken` value.

## Login and save access token

### Bash

```bash
ACCESS_TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alex@example.com",
    "password": "password123"
  }' | jq -r '.accessToken')
```

### PowerShell

```powershell
$login = Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8080/auth/login" `
  -ContentType "application/json" `
  -Body '{
    "email": "alex@example.com",
    "password": "password123"
  }'

$accessToken = $login.accessToken
```

## GetUser

`GetUser` requires authentication.

A regular `USER` can fetch itself.

An `ADMIN` can fetch any user.

### Bash

```bash
grpcurl -plaintext \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{"id": 1}' \
  localhost:50051 \
  api.user.v1.UserService/GetUser
```

### PowerShell

```powershell
grpcurl -plaintext `
  -H "Authorization: Bearer $accessToken" `
  -d '{"id": 1}' `
  localhost:50051 `
  api.user.v1.UserService/GetUser
```

Example response:

```json
{
  "user": {
    "id": "1",
    "name": "Alex",
    "email": "alex@example.com",
    "age": 25,
    "role": "USER",
    "createdAt": "2026-08-24T10:00:00Z",
    "updatedAt": "2026-08-24T10:00:00Z"
  }
}
```

Note: int64 values may be rendered as JSON strings by protobuf JSON encoding.

## ListUsers

`ListUsers` requires `ADMIN` role.

A regular `USER` receives:

```text
PermissionDenied
```

To test `ListUsers` locally, promote the test user to `ADMIN` directly in the local database:

```bash
docker exec identity-service-postgres \
  psql -U identity_service -d identity_service \
  -c "update users set role = 'ADMIN' where email = 'alex@example.com';"
```

Then login again to receive a new access token with the updated role.

### Bash

```bash
ACCESS_TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alex@example.com",
    "password": "password123"
  }' | jq -r '.accessToken')
```

### PowerShell

```powershell
$login = Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8080/auth/login" `
  -ContentType "application/json" `
  -Body '{
    "email": "alex@example.com",
    "password": "password123"
  }'

$accessToken = $login.accessToken
```

Call `ListUsers`:

### Bash

```bash
grpcurl -plaintext \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{
    "limit": 20,
    "offset": 0,
    "email": "",
    "sort": "id_asc"
  }' \
  localhost:50051 \
  api.user.v1.UserService/ListUsers
```

### PowerShell

```powershell
grpcurl -plaintext `
  -H "Authorization: Bearer $accessToken" `
  -d '{
    "limit": 20,
    "offset": 0,
    "email": "",
    "sort": "id_asc"
  }' `
  localhost:50051 `
  api.user.v1.UserService/ListUsers
```

Example response:

```json
{
  "users": [
    {
      "id": "1",
      "name": "Alex",
      "email": "alex@example.com",
      "age": 25,
      "role": "ADMIN",
      "createdAt": "2026-08-24T10:00:00Z",
      "updatedAt": "2026-08-24T10:00:00Z"
    }
  ],
  "total": "1"
}
```

## Calling without reflection

If reflection is disabled, pass the proto file explicitly:

```bash
grpcurl -plaintext \
  -import-path proto \
  -proto api/user/v1/user_service.proto \
  localhost:50051 \
  describe api.user.v1.UserService
```

Get user without reflection:

```bash
grpcurl -plaintext \
  -import-path proto \
  -proto api/user/v1/user_service.proto \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{"id": 1}' \
  localhost:50051 \
  api.user.v1.UserService/GetUser
```

## Common errors

### Unauthenticated

Reason:

```text
Missing, empty, malformed or invalid JWT token.
```

Example:

```text
ERROR:
  Code: Unauthenticated
  Message: unauthenticated
```

### PermissionDenied

Reason:

```text
The token is valid, but the user does not have enough permissions.
```

Example:

```text
ERROR:
  Code: PermissionDenied
  Message: permission denied
```

### InvalidArgument

Reason:

```text
Invalid request payload, for example id <= 0 or invalid pagination parameters.
```

Example:

```text
ERROR:
  Code: InvalidArgument
  Message: invalid user id
```

### NotFound

Reason:

```text
Requested user does not exist.
```

Example:

```text
ERROR:
  Code: NotFound
  Message: User not found
```

## Notes

- gRPC reflection is enabled only in `local` environment.
- `GetUser` allows self-access or `ADMIN` role.
- `ListUsers` requires `ADMIN` role.
- The gRPC API uses the same JWT access token as REST.
- The gRPC server exposes Prometheus metrics through the main HTTP `/metrics` endpoint.