# Setting Up Web App OAuth Client with Localhost

## Step 1: Create/Edit Web Application OAuth Client

1. Go to [Google Cloud Console Credentials](https://console.cloud.google.com/apis/credentials)

2. **If you need to create a new one:**
   - Click "Create Credentials" > "OAuth client ID"
   - Select **"Web application"** as the application type
   - Name it (e.g., "iCloud Photo Sync")
   - Click "Create"

3. **If editing an existing one:**
   - Find your OAuth 2.0 Client ID
   - Click the pencil icon (✏️) to edit

## Step 2: Add Localhost Redirect URI

1. In the OAuth client settings, find the **"Authorized redirect URIs"** section
2. Click **"+ ADD URI"** button
3. Enter exactly:
   ```
   http://localhost:8080
   ```
4. Click "ADD" or press Enter
5. **Important:** Scroll down and click **"SAVE"** at the bottom of the page

## Step 3: Verify the Redirect URI

After saving, you should see `http://localhost:8080` in the list of Authorized redirect URIs.

## Step 4: Get Your Refresh Token

Now you can run the localhost script:

```bash
python3 get_refresh_token_localhost.py
```

This script will:
1. Start a local web server on port 8080
2. Open your browser to the authorization URL
3. After you authorize, it will automatically catch the redirect
4. Exchange the code for a refresh token
5. Display and save your refresh token

## Troubleshooting

- **Port 8080 already in use?** The script will fail. Close any application using port 8080, or modify the script to use a different port (and update the redirect URI in Google Cloud Console to match)

- **Still getting "invalid_request"?** 
  - Make sure you saved the redirect URI in Google Cloud Console
  - Wait 1-2 minutes for changes to propagate
  - Verify the redirect URI is exactly `http://localhost:8080` (no trailing slash, correct port)

- **Browser doesn't open?** Copy the URL from the terminal and paste it into your browser manually

