# Provider setup guide

This guide collects the connection steps for each cloud provider supported by Awd DriveRouter.
Use this as the single reference when you need to set up a new account or troubleshoot a connection.

## Where to configure credentials

Most providers are configured from the app UI under Cloud Accounts or the provider-specific connection modal.
OAuth-based providers also require a callback URL in the provider console:

`http://localhost:5998/oauth/callback`

If you change the app port in the future, update the redirect URI in both the provider console and the app settings.

## Quick reference

| Provider | Connection type | Where to enter credentials | Notes |
| --- | --- | --- | --- |
| Google Drive | OAuth | Settings -> Google Drive Credentials | Uses OAuth client ID and secret |
| OneDrive | OAuth | Settings -> OneDrive / MS Graph Credentials | Uses Microsoft Entra app registration |
| Dropbox | OAuth | Settings -> Dropbox Credentials | Uses Dropbox app key and secret |
| Box | OAuth | Settings -> Box Credentials | Uses Box app credentials |
| Yandex Disk | OAuth | Settings -> Yandex Disk Credentials | Uses Yandex OAuth app credentials |
| pCloud | Direct login | Cloud Accounts -> Connect pCloud | Uses account email and password |
| WebDAV | Direct login | Cloud Accounts -> Connect WebDAV | Uses server URL, username, and password/token |
| S3-compatible storage | Access keys | Cloud Accounts -> Connect S3 | Uses endpoint, bucket, access key, and secret key |
| Telegram Bot | Bot token | Cloud Accounts -> Connect Telegram | Uses bot token and chat ID/channel ID |
| Telegram User | MTProto login | Cloud Accounts -> Connect Telegram User | Uses phone number, API ID, API hash, and verification code |

## Google Drive

1. Open Google Cloud Console.
2. Create or select a project.
3. Enable the Google Drive API.
4. Create an OAuth client ID for a Web application.
5. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

6. Copy the client ID and client secret into the app settings.
7. Open Cloud Accounts and connect Google Drive.

### Notes

- If the app says OAuth is not configured, confirm the client ID and secret are saved in Settings.
- Re-authenticate if Google revokes the token or the token expires.

## OneDrive

1. Open the Microsoft Entra app registration page.
2. Register a new application.
3. Set the supported account type according to your tenant needs.
4. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

5. Add Microsoft Graph delegated permissions for files and profile access.
6. Create a client secret.
7. Save the client ID and secret in the app settings.
8. Open Cloud Accounts and connect OneDrive.

### Notes

- If your tenant requires admin consent, grant it in the Entra portal.
- If login fails, verify that the redirect URI exactly matches the URI configured in the app.

## Dropbox

1. Open the Dropbox App Console.
2. Create a new app.
3. Choose the app permissions needed for file read and write.
4. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

5. Copy the app key and app secret.
6. Save them in the Dropbox credentials section.
7. Open Cloud Accounts and connect Dropbox.

### Notes

- Dropbox credentials are used only for the OAuth flow.
- If the account does not connect, check the app permissions and redirect URI.

## Box

1. Open the Box Developer Console.
2. Create a custom app.
3. Enable OAuth 2.0 user authentication.
4. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

5. Make sure file read and write scopes are enabled.
6. Copy the client ID and client secret into the Box credentials section.
7. Open Cloud Accounts and connect Box.

### Notes

- Box often requires the exact redirect URI and matching scopes.
- If the token exchange fails, re-check the app credential values.

## Yandex Disk

1. Open the Yandex OAuth app console.
2. Create a new application.
3. Choose the Web services platform.
4. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

5. Enable the Yandex.Disk permissions for file access.
6. Save the client ID and client secret in the Yandex Disk credentials section.
7. Open Cloud Accounts and connect Yandex Disk.

### Notes

- Yandex uses OAuth like Google, OneDrive, Dropbox, and Box.
- If the account does not appear after login, confirm the redirect URI and permissions.

## pCloud

pCloud does not require a developer OAuth app in this project.

1. Open Cloud Accounts.
2. Choose Connect -> pCloud.
3. Enter your pCloud email and password.
4. Submit the form.

### Notes

- No client ID or client secret is required for pCloud.
- If you use 2FA, you may need an app-specific password.
- If the login is rejected, confirm the region and account password are correct.

## WebDAV

WebDAV is connected directly from the app UI.

1. Open Cloud Accounts.
2. Choose Connect -> WebDAV.
3. Enter the server URL.
4. Enter the username.
5. Enter the password or app token.
6. Submit the form.

### Notes

- WebDAV does not use OAuth in this app.
- Make sure the server URL includes the correct WebDAV path.
- If the server uses a self-signed certificate, confirm your server trust settings first.

## S3-compatible storage

S3-compatible providers all use the same connection form.

1. Open Cloud Accounts.
2. Choose Connect -> S3.
3. Enter the endpoint URL.
4. Enter the bucket name.
5. Enter the access key ID.
6. Enter the secret access key.
7. Submit the form.

### Notes

- No OAuth client is required.
- The form works with any S3-compatible backend that exposes the S3 API.
- If the connection fails, verify the endpoint, bucket, and credentials.

## Telegram Bot

1. Open Cloud Accounts.
2. Choose Connect -> Telegram.
3. Enter the bot token.
4. Enter the chat ID or channel ID.
5. Submit the form.

### Notes

- This is for bot-based storage access.
- The bot must have access to the destination chat or channel.

## Telegram User

Telegram User uses the MTProto login flow.

1. Open Cloud Accounts.
2. Choose Connect -> Telegram User.
3. Enter your phone number.
4. Enter the Telegram API ID.
5. Enter the Telegram API hash.
6. Request and enter the verification code.
7. If prompted, enter your 2-step verification password.

### Notes

- You must create the Telegram API ID and API hash at `my.telegram.org`.
- This mode stores a session for later reuse.
- Keep the session data and API credentials private.

## Troubleshooting

- If OAuth login opens but fails on callback, verify the redirect URI exactly matches `http://localhost:5998/oauth/callback`.
- If a provider says credentials are missing, check the Settings page first.
- If a direct-login provider fails, confirm the username, password, token, or endpoint manually.
- If the provider does not appear in the app, make sure the app was rebuilt after editing the frontend or Go backend.

## Suggested workflow

1. Read the quick reference table.
2. Open the matching provider section.
3. Configure the provider console or app form.
4. Save credentials.
5. Connect the account from Cloud Accounts.
6. Run sync and confirm the account appears in the file explorer.
