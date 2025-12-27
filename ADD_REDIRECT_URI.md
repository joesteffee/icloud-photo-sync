# How to Add Redirect URI in Google Cloud Console

## Step-by-Step Instructions:

1. **Go to Google Cloud Console**
   - Visit: https://console.cloud.google.com/
   - Make sure you're in the correct project (the one where you created your OAuth client)

2. **Navigate to Credentials**
   - Click on the hamburger menu (☰) in the top left
   - Go to: **APIs & Services** > **Credentials**
   - OR directly visit: https://console.cloud.google.com/apis/credentials

3. **Find Your OAuth 2.0 Client**
   - In the "OAuth 2.0 Client IDs" section, find your client
   - It should show your Client ID: `173564844408-cvat38hiq0ve6032nn7dieu2oo6hji1r.apps.googleusercontent.com`
   - Click on the **pencil icon (✏️)** or the client name to edit it

4. **Add the Redirect URI**
   - Scroll down to the "Authorized redirect URIs" section
   - Click the **"+ ADD URI"** button
   - In the text field that appears, enter:
     ```
     urn:ietf:wg:oauth:2.0:oob
     ```
   - Click **"ADD"** or press Enter

5. **Save Changes**
   - Scroll to the bottom of the page
   - Click the **"SAVE"** button
   - Wait for the confirmation message

6. **Verify**
   - The redirect URI should now appear in the list under "Authorized redirect URIs"
   - You should see: `urn:ietf:wg:oauth:2.0:oob`

## Alternative: If You Want to Use localhost Instead

If you prefer to use `http://localhost:8080` (which works with the localhost script):

1. Follow steps 1-3 above
2. In step 4, instead of `urn:ietf:wg:oauth:2.0:oob`, add:
   ```
   http://localhost:8080
   ```
3. Save and verify

## Troubleshooting

- **Can't find the OAuth client?** Make sure you're in the correct Google Cloud project
- **Don't see "Authorized redirect URIs" section?** Make sure you're editing an OAuth 2.0 Client ID, not an API key
- **Changes not saving?** Make sure you clicked "SAVE" at the bottom of the page

## After Adding the Redirect URI

Once you've added the redirect URI and saved:
1. Wait a minute for the changes to propagate
2. Try the authorization flow again
3. The "invalid_request" error should be resolved

