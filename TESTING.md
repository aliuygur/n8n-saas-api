# Subscription API Testing Guide

## Quick Start

### 1. First, authenticate via Google OAuth:

```bash
# Get the OAuth URL
curl http://127.0.0.1:4000/auth/google/login | jq -r '.auth_url'
```

Open the URL in your browser, complete Google login, and you'll get redirected to:
```
http://localhost:8080/auth/google/callback?code=XXXXX&state=XXXXX
```

Copy the `code` parameter and exchange it for a session token:

```bash
curl -X POST http://127.0.0.1:4000/auth/google/callback \
  -H "Content-Type: application/json" \
  -d '{
    "code": "YOUR_CODE_HERE",
    "state": "YOUR_STATE_HERE"
  }' | jq .
```

Save the `session_token` from the response!

---

## Testing the Subscription Flow

### 2. Create your first instance (starts trial):

```bash
export TOKEN="your_session_token_here"

curl -X POST http://127.0.0.1:4000/me/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subdomain": "my-first-instance"
  }' | jq .
```

**Expected:** ✅ Instance created, trial started for 3 days

### 3. Try to create a second instance (should fail):

```bash
curl -X POST http://127.0.0.1:4000/me/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subdomain": "my-second-instance"
  }' | jq .
```

**Expected:** ❌ Error: "Subscribe to add more instances. Trial users are limited to 1 instance."

### 4. List your instances:

```bash
curl -X GET http://127.0.0.1:4000/me/instances \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 5. Get instance details:

```bash
curl -X GET http://127.0.0.1:4000/me/instances/INSTANCE_ID \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Testing Subscription Activation

### 6. Simulate subscription checkout webhook:

First, get your user_id from the database or from creating an instance.

```bash
curl -X POST http://127.0.0.1:4000/subscription/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "event": "checkout.session.completed",
    "data": {
      "customer_id": "cus_test123",
      "subscription_id": "sub_test123",
      "metadata": {
        "user_id": "YOUR_USER_UUID_HERE"
      }
    }
  }' | jq .
```

**Expected:** ✅ Subscription activated, status changed from "trial" to "active"

### 7. Now try creating a second instance (should succeed):

```bash
curl -X POST http://127.0.0.1:4000/me/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subdomain": "my-second-instance"
  }' | jq .
```

**Expected:** ✅ Second instance created (active subscription allows unlimited instances)

### 8. Delete an instance:

```bash
curl -X DELETE http://127.0.0.1:4000/me/instances/INSTANCE_ID \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Check Database State

```bash
# Connect to the database
encore db shell subscription

# Check subscriptions
SELECT user_id, status, instance_count, trial_ends_at, polar_subscription_id 
FROM subscriptions;

# Exit
\q
```

---

## Testing Trial Expiration

The cron job runs every hour. To test it manually:

```bash
# First, update a trial to be expired in the database
encore db shell subscription

UPDATE subscriptions 
SET trial_ends_at = NOW() - INTERVAL '1 hour' 
WHERE status = 'trial';

\q
```

Then wait for the cron job to run (or trigger it manually if Encore supports that).

---

## API Endpoints Summary

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/auth/google/login` | Public | Get OAuth URL |
| POST | `/auth/google/callback` | Public | Exchange code for token |
| GET | `/auth/me` | Auth | Get current user |
| POST | `/auth/logout` | Auth | Logout |
| POST | `/me/instances` | Auth | Create instance |
| GET | `/me/instances` | Auth | List instances |
| GET | `/me/instances/:id` | Auth | Get instance |
| DELETE | `/me/instances/:id` | Auth | Delete instance |
| POST | `/subscription/webhook` | Public | Polar webhooks |
| POST | `/subscription/cron/expire-trials` | Private | Cron job |

---

## Expected Flow

1. **New User:** No subscription
   - ✅ Can create 1st instance → Trial starts (3 days)
   - ❌ Cannot create 2nd instance → Blocked

2. **Trial User (within 3 days):**
   - ✅ Can use their 1 instance
   - ❌ Cannot create more instances
   - ✅ Can delete and recreate instances

3. **Trial Expired (after 3 days):**
   - ❌ Cannot create new instances
   - ❌ Existing instances may be suspended

4. **Active Subscription:**
   - ✅ Can create unlimited instances
   - 💰 Pays $10/month per instance
   - ✅ Prorated billing when adding/removing instances

---

## Troubleshooting

If you see errors:

1. **"user not authenticated"** → Token expired or invalid
2. **"Subscribe to add more instances"** → Working as intended! Trial limit enforced
3. **"Your trial has expired"** → Need to activate subscription
4. **Database errors** → Check migrations ran: `encore db reset --all`

---

## Next: Create Real Polar Checkout

```bash
# This would be called from frontend when user clicks "Subscribe"
# Returns a Polar checkout URL to redirect user to
curl -X POST http://127.0.0.1:4000/subscription/create-checkout \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "YOUR_USER_UUID",
    "success_url": "http://localhost:8080/dashboard?subscribed=true",
    "return_url": "http://localhost:8080/dashboard"
  }'
```

This is a private endpoint, so you'll need to call it from the API service or add it to frontend integration.
