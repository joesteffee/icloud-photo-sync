
#!/bin/bash
# Script to exchange authorization code for refresh token
# Usage: ./exchange_token.sh YOUR_AUTH_CODE

AUTH_CODE="$1"
CLIENT_ID="${GOOGLE_PHOTOS_CLIENT_ID:-YOUR_CLIENT_ID_HERE}"
CLIENT_SECRET="${GOOGLE_PHOTOS_CLIENT_SECRET:-YOUR_CLIENT_SECRET_HERE}"
REDIRECT_URI="${GOOGLE_PHOTOS_REDIRECT_URI:-http://localhost:8080}"

if [ "$CLIENT_ID" = "YOUR_CLIENT_ID_HERE" ] || [ "$CLIENT_SECRET" = "YOUR_CLIENT_SECRET_HERE" ]; then
    echo "ERROR: Please set GOOGLE_PHOTOS_CLIENT_ID and GOOGLE_PHOTOS_CLIENT_SECRET environment variables"
    echo "Or edit this script to replace YOUR_CLIENT_ID_HERE and YOUR_CLIENT_SECRET_HERE"
    exit 1
fi

if [ -z "$AUTH_CODE" ]; then
    echo "Usage: $0 YOUR_AUTH_CODE"
    exit 1
fi

echo "Exchanging authorization code for tokens..."
echo ""

curl -X POST https://oauth2.googleapis.com/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "code=$AUTH_CODE" \
  -d "client_id=$CLIENT_ID" \
  -d "client_secret=$CLIENT_SECRET" \
  -d "redirect_uri=$REDIRECT_URI" \
  -d "grant_type=authorization_code" | python3 -m json.tool

echo ""
echo "Look for 'refresh_token' in the output above."

