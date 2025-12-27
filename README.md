# iCloud Photo Frame Sync Service

A Go-based Docker service that periodically scrapes iCloud shared albums, downloads unique images, tracks them in Redis by content hash, emails new photos to a digital photo frame, and optionally syncs them to a Google Photos album.

## Features

- Scrapes iCloud shared album public URLs for photos
- Downloads and stores unique images in a mounted directory
- Tracks processed images by content hash in Redis (separately for email and Google Photos)
- Emails new photos to a specified email address
- Uploads new photos to a Google Photos album (optional)
- Runs continuously on a configurable interval
- Limits number of photos processed per run (applies to both email and Google Photos)

## Requirements

- Go 1.18+
- Docker (for containerized deployment)
- Redis server
- SMTP server access
- Google Cloud Project with Photos Library API enabled (optional, for Google Photos sync)

## Configuration

### Album Configuration File

The service reads album URLs from a JSON configuration file located at `/images/config.json` (or `${IMAGE_DIR}/config.json` if `IMAGE_DIR` is set). This file should contain a list of iCloud shared album URLs to sync from.

**Example configuration file (`/images/config.json`):**

```json
{
  "album_urls": [
    "https://www.icloud.com/sharedalbum/#EXAMPLE_TOKEN",
    "https://www.icloud.com/sharedalbum/#A1Y48TkBrRUFpV",
    "https://www.icloud.com/sharedalbum/#C3A60VmDsTUGrX"
  ]
}
```

You can specify multiple album URLs in the `album_urls` array. The service will sync images from all specified albums.

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `REDIS_URL` | Redis connection URL (e.g., `redis://localhost:6379`) | Yes | - |
| `SMTP_SERVER` | SMTP server hostname | Yes | - |
| `SMTP_PORT` | SMTP server port | Yes | - |
| `SMTP_USERNAME` | SMTP username | Yes | - |
| `SMTP_PASSWORD` | SMTP password | Yes | - |
| `SMTP_FROM` | Email address for Reply-To header. The "From" header will always use `SMTP_USERNAME` to match the authenticated user (required by some SMTP servers like ProtonMail Bridge). | No | `SMTP_USERNAME` |
| `SMTP_DESTINATION` | Email address to send photos to | Yes | - |
| `RUN_INTERVAL` | Seconds between runs (applies to both email and Google Photos) | No | 3600 |
| `MAX_ITEMS` | Maximum number of new photos to process per run (applies to both email and Google Photos) | No | 5 |
| `IMAGE_DIR` | Directory to store downloaded images and config file | No | `/images` |
| `GOOGLE_PHOTOS_CLIENT_ID` | OAuth2 client ID for Google Photos API | No* | - |
| `GOOGLE_PHOTOS_CLIENT_SECRET` | OAuth2 client secret for Google Photos API | No* | - |
| `GOOGLE_PHOTOS_REFRESH_TOKEN` | OAuth2 refresh token for Google Photos API | No* | - |
| `GOOGLE_PHOTOS_ALBUM_NAME` | Name of the pre-existing Google Photos album to upload to | No* | - |

\* Google Photos environment variables are optional. If any are provided, all must be provided. See [Setting Up Google Photos](#setting-up-google-photos) for detailed instructions.

## Usage

### Docker

1. Create the configuration file:
   ```bash
   mkdir -p /path/to/images
   cat > /path/to/images/config.json << EOF
   {
     "album_urls": [
       "https://www.icloud.com/sharedalbum/#EXAMPLE_TOKEN",
       "https://www.icloud.com/sharedalbum/#A1Y48TkBrRUFpV"
     ]
   }
   EOF
   ```

2. Build the Docker image:
   ```bash
   docker build -t icloud-photo-sync:latest .
   ```

3. Run the container:
   ```bash
   docker run -d \
     -e REDIS_URL="redis://redis:6379" \
     -e SMTP_SERVER="smtp.gmail.com" \
     -e SMTP_PORT="587" \
     -e SMTP_USERNAME="your-email@gmail.com" \
     -e SMTP_PASSWORD="your-password" \
     -e SMTP_FROM="photoframe@example.com" \
     -e SMTP_DESTINATION="photo-frame@example.com" \
     -e RUN_INTERVAL="3600" \
     -e MAX_ITEMS="5" \
     -v /path/to/images:/images \
     icloud-photo-sync:latest
   ```

   **Note:** Some SMTP servers (like ProtonMail Bridge) require the "From" address to match the authenticated username. In this case, the service will use `SMTP_USERNAME` as the "From" address and `SMTP_FROM` (if provided) as the "Reply-To" header. The service also supports self-signed certificates and will skip certificate verification when needed.

   **With Google Photos (optional):**
   ```bash
   docker run -d \
     -e REDIS_URL="redis://redis:6379" \
     -e SMTP_SERVER="smtp.gmail.com" \
     -e SMTP_PORT="587" \
     -e SMTP_USERNAME="your-email@gmail.com" \
     -e SMTP_PASSWORD="your-password" \
     -e SMTP_DESTINATION="photo-frame@example.com" \
     -e GOOGLE_PHOTOS_CLIENT_ID="your-client-id" \
     -e GOOGLE_PHOTOS_CLIENT_SECRET="your-client-secret" \
     -e GOOGLE_PHOTOS_REFRESH_TOKEN="your-refresh-token" \
     -e GOOGLE_PHOTOS_ALBUM_NAME="My Photo Album" \
     -e RUN_INTERVAL="3600" \
     -e MAX_ITEMS="5" \
     -v /path/to/images:/images \
     icloud-photo-sync:latest
   ```

### Local Go

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Create the configuration file:
   ```bash
   mkdir -p ./images
   cat > ./images/config.json << EOF
   {
     "album_urls": [
       "https://www.icloud.com/sharedalbum/#EXAMPLE_TOKEN",
       "https://www.icloud.com/sharedalbum/#A1Y48TkBrRUFpV"
     ]
   }
   EOF
   ```

3. Set environment variables and run:
   ```bash
   export REDIS_URL="redis://localhost:6379"
   export SMTP_SERVER="smtp.gmail.com"
   export SMTP_PORT="587"
   export SMTP_USERNAME="your-email@gmail.com"
   export SMTP_PASSWORD="your-password"
   export SMTP_DESTINATION="photo-frame@example.com"
   export RUN_INTERVAL="3600"
   export MAX_ITEMS="5"
   export IMAGE_DIR="./images"
   # Optional: Add Google Photos configuration
   export GOOGLE_PHOTOS_CLIENT_ID="your-client-id"
   export GOOGLE_PHOTOS_CLIENT_SECRET="your-client-secret"
   export GOOGLE_PHOTOS_REFRESH_TOKEN="your-refresh-token"
   export GOOGLE_PHOTOS_ALBUM_NAME="My Photo Album"
   go run main.go
   ```

## How It Works

1. **Scraping**: The service fetches the iCloud shared album page and extracts image URLs from the HTML/JavaScript content.

2. **Download & Hash**: For each image URL:
   - Downloads the image to the configured directory
   - Calculates SHA-256 hash of the image content
   - Checks if the hash already exists in Redis (for email tracking)

3. **Processing New Photos**: For new images (not yet processed for email):
   - **Email**: Emails the image as an attachment to the configured destination
   - **Google Photos**: Uploads the image to the specified Google Photos album (if configured)
   - Respects the `MAX_ITEMS` limit per run (applies to both services)
   - Both services process the same new photos in parallel

4. **Tracking**: After successful processing:
   - Stores the image hash in Redis separately for email and Google Photos tracking
   - This allows independent tracking - a photo can be emailed but not yet uploaded to Google Photos (or vice versa)
   - Keeps the image file in the mounted directory

5. **Continuous Operation**: The service runs in a loop, waiting `RUN_INTERVAL` seconds between each check for new photos.

## Setting Up Google Photos

To enable Google Photos sync, you need to set up OAuth2 credentials and create a Google Photos album. Follow these steps:

### Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google Photos Library API:
   - Navigate to "APIs & Services" > "Library"
   - Search for "Photos Library API"
   - Click on it and press "Enable"

### Step 2: Create OAuth 2.0 Credentials

1. In Google Cloud Console, go to "APIs & Services" > "Credentials"
2. Click "Create Credentials" > "OAuth client ID"
3. If prompted, configure the OAuth consent screen:
   - Choose "External" user type (unless you have a Google Workspace account)
   - Fill in the required fields:
     - App name: e.g., "iCloud Photo Sync"
     - User support email: Your email address
     - Developer contact information: Your email address
   - Click "Save and Continue"
   - Add scopes:
     - `https://www.googleapis.com/auth/photoslibrary`
     - `https://www.googleapis.com/auth/photoslibrary.appendonly`
   - Click "Save and Continue"
   - Add test users (your Google account email) if the app is in testing mode
   - Click "Save and Continue" then "Back to Dashboard"
4. For application type, choose "Desktop app" or "Other"
5. Name the OAuth client (e.g., "iCloud Photo Sync")
6. **Important:** When creating a Desktop app OAuth client, the redirect URI will be automatically set to `urn:ietf:wg:oauth:2.0:oob` (which is what the script uses). If you choose "Other", you may need to add this redirect URI manually.
7. Click "Create"
8. **Save the Client ID and Client Secret** - you'll need these for `GOOGLE_PHOTOS_CLIENT_ID` and `GOOGLE_PHOTOS_CLIENT_SECRET`

### Step 3: Obtain Refresh Token

**Important:** The refresh token is required because access tokens expire after a short time (typically 1 hour). The refresh token allows the service to automatically get new access tokens without requiring user interaction, which is essential for a long-running service.

**Note:** The OAuth 2.0 Playground does not include the Photos Library API in its list of available APIs, so you'll need to use one of the methods below.

#### Option A: Using a Python Script (Recommended)

Save this Python script and run it to obtain your refresh token:

```python
#!/usr/bin/env python3
import urllib.parse
import urllib.request
import json

# Replace these with your values from Step 2
CLIENT_ID = "your-client-id"
CLIENT_SECRET = "your-client-secret"
REDIRECT_URI = "urn:ietf:wg:oauth:2.0:oob"  # For desktop apps

# Step 1: Construct authorization URL
scopes = [
    "https://www.googleapis.com/auth/photoslibrary",
    "https://www.googleapis.com/auth/photoslibrary.appendonly"
]
scope_string = "+".join(scopes)

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

print("=" * 60)
print("Step 1: Open this URL in your browser:")
print("=" * 60)
print(auth_url)
print("\nStep 2: Sign in with your Google account and authorize the application")
print("Step 3: After authorization, you'll be redirected to a page showing an authorization code")
print("Step 4: Copy the authorization code from that page")
print("=" * 60)

auth_code = input("\nEnter the authorization code: ")

# Step 2: Exchange authorization code for tokens
token_url = "https://oauth2.googleapis.com/token"
data = urllib.parse.urlencode({
    "code": auth_code,
    "client_id": CLIENT_ID,
    "client_secret": CLIENT_SECRET,
    "redirect_uri": REDIRECT_URI,
    "grant_type": "authorization_code"
}).encode()

req = urllib.request.Request(token_url, data=data, headers={"Content-Type": "application/x-www-form-urlencoded"})
response = urllib.request.urlopen(req)
tokens = json.loads(response.read().decode())

print("\n" + "=" * 60)
print("SUCCESS! Your refresh token:")
print("=" * 60)
print(tokens["refresh_token"])
print("\nSave this as your GOOGLE_PHOTOS_REFRESH_TOKEN")
print("=" * 60)
```

**Important:** Before running the script:
1. Make sure your OAuth client in Google Cloud Console has the redirect URI `urn:ietf:wg:oauth:2.0:oob` configured (or update the script to use a different redirect URI that matches your OAuth client settings)
2. Replace `your-client-id` and `your-client-secret` with your actual values from Step 2
3. The `prompt=consent` parameter ensures you get a refresh token even if you've authorized the app before

#### Option B: Manual OAuth Flow

If you prefer not to use Python, you can do this manually:

1. **Construct the authorization URL:**
   ```
   https://accounts.google.com/o/oauth2/v2/auth?client_id=YOUR_CLIENT_ID&redirect_uri=urn:ietf:wg:oauth:2.0:oob&response_type=code&scope=https://www.googleapis.com/auth/photoslibrary+https://www.googleapis.com/auth/photoslibrary.appendonly&access_type=offline&prompt=consent
   ```
   Replace `YOUR_CLIENT_ID` with your actual client ID.

2. **Open the URL in your browser**, sign in, and authorize the application.

3. **Copy the authorization code** from the redirect page.

4. **Exchange the code for tokens** using curl:
   ```bash
   curl -X POST https://oauth2.googleapis.com/token \
     -d "code=AUTHORIZATION_CODE" \
     -d "client_id=YOUR_CLIENT_ID" \
     -d "client_secret=YOUR_CLIENT_SECRET" \
     -d "redirect_uri=urn:ietf:wg:oauth:2.0:oob" \
     -d "grant_type=authorization_code"
   ```
   Replace `AUTHORIZATION_CODE`, `YOUR_CLIENT_ID`, and `YOUR_CLIENT_SECRET` with your actual values.

5. **Extract the refresh_token** from the JSON response.

Save this script, replace `CLIENT_ID` and `CLIENT_SECRET` with your values, run it, and follow the prompts.

### Step 4: Create a Google Photos Album

1. Open [Google Photos](https://photos.google.com/) in a web browser
2. Navigate to the "Albums" section (left sidebar)
3. Click the "+" button to create a new album
4. Name your album (e.g., "iCloud Sync Photos")
5. **Important:** Remember the exact album name - it must match exactly (case-sensitive, including spaces and special characters)
6. This name is your `GOOGLE_PHOTOS_ALBUM_NAME`

**Note:** The album must exist before running the service. The service will look up the album by name and fail if it doesn't exist.

### Step 5: Configure Environment Variables

Set all four Google Photos environment variables:

```bash
export GOOGLE_PHOTOS_CLIENT_ID="your-client-id-from-step-2"
export GOOGLE_PHOTOS_CLIENT_SECRET="your-client-secret-from-step-2"
export GOOGLE_PHOTOS_REFRESH_TOKEN="your-refresh-token-from-step-3"
export GOOGLE_PHOTOS_ALBUM_NAME="your-album-name-from-step-4"
```

**Important:** If any Google Photos environment variable is set, all four must be set, or the service will fail to start.

## Troubleshooting

### Google Photos Issues

- **"Album not found" error**: Verify the album name matches exactly (case-sensitive). Check the album name in Google Photos and ensure there are no extra spaces.
- **"Invalid credentials" error**: Verify your Client ID, Client Secret, and Refresh Token are correct. Make sure the OAuth consent screen is properly configured.
- **"API not enabled" error**: Ensure the Photos Library API is enabled in your Google Cloud project.
- **Token refresh failures**: Refresh tokens don't expire unless revoked. If you get token errors, you may need to generate a new refresh token using the steps above.

### General Issues

- Images are identified by their content hash (SHA-256), not by URL, to handle cases where URLs might change but content is the same
- The service is smart about re-downloading: it only downloads new URLs or when hash verification is needed
- All images are stored in the mounted directory for persistence
- The service gracefully handles errors and continues running even if individual operations fail
- Email and Google Photos sync status are tracked separately in Redis, so a photo can be emailed but not yet uploaded to Google Photos (or vice versa)

## Notes

- Images are identified by their content hash (SHA-256), not by URL, to handle cases where URLs might change but content is the same
- The service is smart about re-downloading: it only downloads new URLs or when hash verification is needed
- All images are stored in the mounted directory for persistence
- The service gracefully handles errors and continues running even if individual operations fail
- Email and Google Photos sync status are tracked separately in Redis
- Same photos are sent to both email and Google Photos if they're new
- Same `MAX_ITEMS` and `RUN_INTERVAL` settings apply to both services
- If Google Photos is not configured, only email functionality runs (backward compatible)

## License

This project is provided as-is for personal use.

