# Keycloak Setup Guide

Keycloak 26.2 setup for the book list app.

## Prerequisites

Docker and `task` must be installed. The Keycloak container is defined in `utils/docker-compose.yaml`.

---

## 1. Start Keycloak

```bash
task kc:up    # start the container
task kc:wait  # wait until it's ready (~30 seconds)
```

Keycloak will be available at **http://localhost:8180**.

---

## 2. Log into the Admin Console

1. Open **http://localhost:8180**
2. Click **Administration Console**
3. Log in with `admin` / `admin`

---

## 3. Create the Realm

1. In the top-left dropdown (showing **Keycloak**), click **Create realm**
2. Set **Realm name** to `booklist`
3. Leave **Enabled** ON
4. Click **Create**

---

## 4. Create the Client

1. In the left sidebar, go to **Clients** → **Create client**
2. **General Settings:**
   - Client type: `OpenID Connect`
   - Client ID: `booklist-api`
   - Click **Next**
3. **Capability Config:**
   - Client authentication: **OFF** (public client)
   - Standard flow: **ON** (needed for PKCE frontend login)
   - Direct access grants: **ON** (needed for Bruno/curl token testing)
   - Click **Next**
4. **Login Settings:**
   - Valid redirect URIs: `http://localhost:5173/*`
   - Web origins: `http://localhost:5173`
   - Click **Save**

---

## 5. Create a Test User

> **Keycloak 26 requires `email`, `firstName`, and `lastName`** — missing any of these causes `"Account is not fully set up"` errors when fetching tokens.

1. In the left sidebar, go to **Users** → **Create new user**
2. Fill in:
   - Username: `testuser`
   - Email: `testuser@example.com`
   - Email verified: **ON**
   - First name: `Test`
   - Last name: `User`
3. Click **Create**
4. Go to the **Credentials** tab
5. Click **Set password**
   - Password: `password`
   - Temporary: **OFF**
6. Click **Save** → **Save password**

---

## 6. Verify It Works

Get a token:

```bash
curl -s -X POST \
  "http://localhost:8180/realms/booklist/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=booklist-api&grant_type=password&username=testuser&password=password" \
  | jq '{access_token: .access_token[0:40], expires_in, token_type}'
```

Expected response:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "expires_in": 300,
  "token_type": "Bearer"
}
```

Use it against the protected API:

```bash
TOKEN=$(curl -s -X POST \
  "http://localhost:8180/realms/booklist/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "client_id=booklist-api&grant_type=password&username=testuser&password=password" \
  | jq -r '.access_token')

curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/readlist/ | jq .
```

---

## 7. Bruno Setup

Open Bruno and load `utils/Bruno/book-api/`. The **GET token** request uses the direct access grant — run it first, then the token is available for the readlist requests.

---

## Common Errors

| Error | Cause | Fix |
|---|---|---|
| `Realm does not exist` | Backend started before realm was created | Complete steps 3–5 first, then restart the backend |
| `Account is not fully set up` | User missing `email`, `firstName`, or `lastName` | Add all three fields when creating the user |
| `401 Unauthorized` from API | Token expired (5 min TTL) | Re-run the GET token request in Bruno |
| `invalid_client` | Wrong client ID | Ensure client ID is exactly `booklist-api` |
