#!/usr/bin/env python3
"""
Script to obtain Google Photos API refresh token using localhost redirect.
This version starts a local server to catch the OAuth redirect.
"""
import urllib.parse
import urllib.request
import json
import webbrowser
import http.server
import socketserver
from urllib.parse import urlparse, parse_qs
import threading

# Your OAuth credentials
# Set these as environment variables or replace the placeholders below
import os
CLIENT_ID = os.getenv("GOOGLE_PHOTOS_CLIENT_ID", "YOUR_CLIENT_ID_HERE")
CLIENT_SECRET = os.getenv("GOOGLE_PHOTOS_CLIENT_SECRET", "YOUR_CLIENT_SECRET_HERE")
REDIRECT_URI = "http://localhost:8080"  # Must match what's configured in Google Cloud Console

if CLIENT_ID == "YOUR_CLIENT_ID_HERE" or CLIENT_SECRET == "YOUR_CLIENT_SECRET_HERE":
    print("ERROR: Please set GOOGLE_PHOTOS_CLIENT_ID and GOOGLE_PHOTOS_CLIENT_SECRET environment variables")
    print("Or edit this script to replace YOUR_CLIENT_ID_HERE and YOUR_CLIENT_SECRET_HERE")
    exit(1)

# Google Photos API scopes
# Note: As of March 2025, Google deprecated the photoslibrary scope
# New scopes only allow access to app-created albums/media
# Scopes must be space-separated in the OAuth URL
scopes = [
    "https://www.googleapis.com/auth/photoslibrary.appendonly",
    "https://www.googleapis.com/auth/photoslibrary.readonly.appcreateddata",
    "https://www.googleapis.com/auth/photoslibrary.edit.appcreateddata",
]
scope_string = " ".join(scopes)  # Space-separated, not plus-separated

# Global variable to store the authorization code
auth_code = None
code_received = threading.Event()

class OAuthHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        global auth_code
        parsed_url = urlparse(self.path)
        query_params = parse_qs(parsed_url.query)
        
        if 'code' in query_params:
            auth_code = query_params['code'][0]
            self.send_response(200)
            self.send_header('Content-type', 'text/html')
            self.end_headers()
            self.wfile.write(b"""
            <html>
            <head><title>Authorization Successful</title></head>
            <body>
            <h1>Authorization Successful!</h1>
            <p>You can close this window and return to the terminal.</p>
            </body>
            </html>
            """)
            code_received.set()
        elif 'error' in query_params:
            error = query_params['error'][0]
            error_desc = query_params.get('error_description', [''])[0]
            self.send_response(400)
            self.send_header('Content-type', 'text/html')
            self.end_headers()
            self.wfile.write(f"""
            <html>
            <head><title>Authorization Failed</title></head>
            <body>
            <h1>Authorization Failed</h1>
            <p>Error: {error}</p>
            <p>{error_desc}</p>
            </body>
            </html>
            """.encode())
        else:
            self.send_response(200)
            self.send_header('Content-type', 'text/html')
            self.end_headers()
            self.wfile.write(b"<html><body>Waiting for authorization...</body></html>")

def start_local_server():
    """Start a local HTTP server to catch the OAuth redirect"""
    port = 8080
    handler = OAuthHandler
    httpd = socketserver.TCPServer(("", port), handler)
    print(f"Local server started on http://localhost:{port}")
    print("Waiting for OAuth redirect...")
    httpd.timeout = 300  # 5 minute timeout
    while not code_received.is_set():
        httpd.handle_request()
    return auth_code

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
print("Google Photos API - Refresh Token Generator (Localhost Method)")
print("=" * 70)
print("\nIMPORTANT: Make sure your OAuth client in Google Cloud Console")
print("has this redirect URI configured:")
print(f"  http://localhost:8080")
print("\nTo add it:")
print("1. Go to Google Cloud Console > APIs & Services > Credentials")
print("2. Click on your OAuth 2.0 Client ID")
print("3. Under 'Authorized redirect URIs', click 'ADD URI'")
print("4. Add: http://localhost:8080")
print("5. Click 'SAVE'")
print("\n" + "=" * 70)
print("\nStep 1: Opening authorization URL in your browser...")
print("If the browser doesn't open automatically, copy this URL:")
print("\n" + auth_url + "\n")

# Start the local server in a thread
server_thread = threading.Thread(target=start_local_server, daemon=True)
server_thread.start()

# Try to open the URL in the browser
try:
    webbrowser.open(auth_url)
    print("Browser opened! Please authorize the application.")
except:
    print("Could not open browser automatically. Please copy the URL above.")

# Wait for the authorization code
print("\nWaiting for authorization...")
code_received.wait(timeout=300)  # 5 minute timeout

if not auth_code:
    print("\nTimeout: No authorization code received.")
    exit(1)

print("\nAuthorization code received! Exchanging for tokens...")

# Step 2: Exchange authorization code for tokens
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
        print("Try revoking access and running this script again.")
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

