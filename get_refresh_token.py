#!/usr/bin/env python3
"""
Script to obtain Google Photos API refresh token.
Run this script and follow the prompts.
"""
import urllib.parse
import urllib.request
import json
import webbrowser

# Your OAuth credentials
# Set these as environment variables or replace the placeholders below
import os
CLIENT_ID = os.getenv("GOOGLE_PHOTOS_CLIENT_ID", "YOUR_CLIENT_ID_HERE")
CLIENT_SECRET = os.getenv("GOOGLE_PHOTOS_CLIENT_SECRET", "YOUR_CLIENT_SECRET_HERE")
REDIRECT_URI = os.getenv("GOOGLE_PHOTOS_REDIRECT_URI", "urn:ietf:wg:oauth:2.0:oob")

if CLIENT_ID == "YOUR_CLIENT_ID_HERE" or CLIENT_SECRET == "YOUR_CLIENT_SECRET_HERE":
    print("ERROR: Please set GOOGLE_PHOTOS_CLIENT_ID and GOOGLE_PHOTOS_CLIENT_SECRET environment variables")
    print("Or edit this script to replace YOUR_CLIENT_ID_HERE and YOUR_CLIENT_SECRET_HERE")
    exit(1)

# Google Photos API scopes
# Scopes must be space-separated in the OAuth URL
scopes = [
    "https://www.googleapis.com/auth/photoslibrary",
    "https://www.googleapis.com/auth/photoslibrary.appendonly"
]
scope_string = " ".join(scopes)  # Space-separated, not plus-separated

# Step 1: Construct authorization URL
auth_url = (
    "https://accounts.google.com/o/oauth2/v2/auth?"
    "client_id={}&"
    "redirect_uri={}&"
    "response_type=code&"
    "scope={}&"
    "access_type=offline&"
    "prompt=consent"
).format(
    CLIENT_ID,
    urllib.parse.quote(REDIRECT_URI),
    urllib.parse.quote(scope_string)
)

print("=" * 70)
print("Google Photos API - Refresh Token Generator")
print("=" * 70)
print("\nIMPORTANT: Before proceeding, make sure your OAuth client in Google Cloud Console")
print("has the redirect URI configured:")
print(f"  Redirect URI: {REDIRECT_URI}")
print("\nIf you get an 'invalid_request' error, you need to:")
print("1. Go to Google Cloud Console > APIs & Services > Credentials")
print("2. Click on your OAuth 2.0 Client ID")
print("3. Add this redirect URI to 'Authorized redirect URIs':")
print(f"   {REDIRECT_URI}")
print("4. Save and try again")
print("\n" + "=" * 70)
print("\nStep 1: Opening authorization URL in your browser...")
print("If the browser doesn't open automatically, copy this URL:")
print("\n" + auth_url + "\n")

# Try to open the URL in the browser
try:
    webbrowser.open(auth_url)
    print("Browser opened! Please authorize the application.")
except:
    print("Could not open browser automatically. Please copy the URL above.")

print("\n" + "=" * 70)
if REDIRECT_URI.startswith("http://localhost"):
    print("Step 2: After authorization, you'll be redirected to localhost")
    print("Step 3: Check the URL in your browser - it will contain 'code=' parameter")
    print("Step 4: Copy everything after 'code=' (before any '&' or end of URL)")
    print("=" * 70)
else:
    print("Step 2: After authorization, you'll see a page with an authorization code")
    print("Step 3: Copy that code and paste it below")
    print("=" * 70)

auth_code = input("\nEnter the authorization code: ").strip()

if not auth_code:
    print("Error: No authorization code provided.")
    exit(1)

# Step 2: Exchange authorization code for tokens
print("\nExchanging authorization code for tokens...")
token_url = "https://oauth2.googleapis.com/token"
data = urllib.parse.urlencode({
    "code": auth_code,
    "client_id": CLIENT_ID,
    "client_secret": CLIENT_SECRET,
    "redirect_uri": REDIRECT_URI,
    "grant_type": "authorization_code"
}).encode()

try:
    req = urllib.request.Request(
        token_url, 
        data=data, 
        headers={"Content-Type": "application/x-www-form-urlencoded"}
    )
    response = urllib.request.urlopen(req)
    tokens = json.loads(response.read().decode())
    
    if "refresh_token" not in tokens:
        print("\n" + "=" * 70)
        print("WARNING: No refresh token in response!")
        print("This might happen if you've already authorized this app before.")
        print("Try revoking access and running this script again, or use")
        print("the 'prompt=consent' parameter (which is already included).")
        print("=" * 70)
        print("\nResponse received:")
        print(json.dumps(tokens, indent=2))
    else:
        print("\n" + "=" * 70)
        print("SUCCESS! Your refresh token:")
        print("=" * 70)
        print(tokens["refresh_token"])
        print("\n" + "=" * 70)
        print("Save this as your GOOGLE_PHOTOS_REFRESH_TOKEN environment variable")
        print("=" * 70)
        
        # Also save to a file for convenience
        with open("refresh_token.txt", "w") as f:
            f.write(tokens["refresh_token"])
        print("\nRefresh token also saved to: refresh_token.txt")
        
except urllib.error.HTTPError as e:
    error_body = e.read().decode()
    print("\n" + "=" * 70)
    print("ERROR: Failed to exchange authorization code")
    print("=" * 70)
    print(f"Status: {e.code}")
    print(f"Response: {error_body}")
    print("=" * 70)
    try:
        error_json = json.loads(error_body)
        if "error_description" in error_json:
            print(f"\nError: {error_json['error_description']}")
    except:
        pass
except Exception as e:
    print("\n" + "=" * 70)
    print("ERROR: An unexpected error occurred")
    print("=" * 70)
    print(str(e))
    print("=" * 70)

