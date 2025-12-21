# iCloud Photo Frame Sync Service

A Go-based Docker service that periodically scrapes iCloud shared albums, downloads unique images, tracks them in Redis by content hash, and emails new photos to a digital photo frame.

## Features

- Scrapes iCloud shared album public URLs for photos
- Downloads and stores unique images in a mounted directory
- Tracks processed images by content hash in Redis
- Emails new photos to a specified email address
- Runs continuously on a configurable interval
- Limits number of emails per run to avoid overwhelming the recipient

## Requirements

- Go 1.18+
- Docker (for containerized deployment)
- Redis server
- SMTP server access

## Configuration

### Album Configuration File

The service reads album URLs from a JSON configuration file located at `/images/config.json` (or `${IMAGE_DIR}/config.json` if `IMAGE_DIR` is set). This file should contain a list of iCloud shared album URLs to sync from.

**Example configuration file (`/images/config.json`):**

```json
{
  "album_urls": [
    "https://www.icloud.com/sharedalbum/#B2Z59UlCqSTGqW",
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
| `RUN_INTERVAL` | Seconds between runs | No | 3600 |
| `MAX_ITEMS` | Maximum number of photos to email per run | No | 5 |
| `IMAGE_DIR` | Directory to store downloaded images and config file | No | `/images` |

## Usage

### Docker

1. Create the configuration file:
   ```bash
   mkdir -p /path/to/images
   cat > /path/to/images/config.json << EOF
   {
     "album_urls": [
       "https://www.icloud.com/sharedalbum/#B2Z59UlCqSTGqW",
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
       "https://www.icloud.com/sharedalbum/#B2Z59UlCqSTGqW",
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
   go run main.go
   ```

## How It Works

1. **Scraping**: The service fetches the iCloud shared album page and extracts image URLs from the HTML/JavaScript content.

2. **Download & Hash**: For each image URL:
   - Downloads the image to the configured directory
   - Calculates SHA-256 hash of the image content
   - Checks if the hash already exists in Redis

3. **Email**: For new images (hash not in Redis):
   - Emails the image as an attachment to the configured destination
   - Respects the `MAX_ITEMS` limit per run
   - Only sends emails for images that haven't been processed

4. **Tracking**: After successful email send:
   - Stores the image hash in Redis to prevent re-processing
   - Keeps the image file in the mounted directory

5. **Continuous Operation**: The service runs in a loop, waiting `RUN_INTERVAL` seconds between each check for new photos.

## Notes

- Images are identified by their content hash (SHA-256), not by URL, to handle cases where URLs might change but content is the same
- The service is smart about re-downloading: it only downloads new URLs or when hash verification is needed
- All images are stored in the mounted directory for persistence
- The service gracefully handles errors and continues running even if individual operations fail

## License

This project is provided as-is for personal use.

