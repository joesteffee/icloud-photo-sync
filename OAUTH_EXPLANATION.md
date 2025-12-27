# OAuth2 Token Explanation

## What Each Token Does:

1. **Client ID & Secret** (what you have):
   - ✅ Never expire
   - Used to identify your application
   - Not enough to access the API - you still need user authorization

2. **Access Token**:
   - ❌ Expires after ~1 hour
   - Used to make actual API calls
   - Cannot be used long-term without refresh

3. **Refresh Token** (what we need):
   - ✅ Never expires (unless you revoke it)
   - Used to get new access tokens automatically
   - **One-time setup** - get it once, use it forever

## Why We Need the Refresh Token:

- Your service runs continuously (every hour, day, etc.)
- Access tokens expire after 1 hour
- Without a refresh token, your service would stop working after 1 hour
- With a refresh token, the service automatically gets new access tokens in the background

## The Good News:

- **You only need to do this ONCE**
- The refresh token lasts forever (unless you revoke it)
- After setup, everything is automatic
- No more user interaction needed

## Google Photos API Limitation:

Unfortunately, Google Photos API **does not support service accounts** (which would allow server-to-server auth without user interaction). OAuth2 with refresh tokens is the only way.

## Simplest Way to Get It:

The easiest method is to:
1. Fix the redirect URI in Google Cloud Console (add `urn:ietf:wg:oauth:2.0:oob`)
2. Visit the authorization URL
3. Copy the code
4. Run the exchange script

Once you have the refresh token, you never need to do this again!

