# Provider credential setup

This guide explains how to create credentials and connect cloud storage providers supported by Awd DriveRouter.

Keep this guide separate from the main `README.md` so the README stays short while provider-specific setup remains detailed.

## Redirect URIs used by Awd DriveRouter

Use this callback URL when running Awd DriveRouter:

| Provider | Redirect URI |
| --- | --- |
| Google Drive, OneDrive, Dropbox, Box, Yandex Disk, pCloud | `http://localhost:5998/oauth/callback` |

If you change the app port in the future, update the redirect URI in both the provider console and the app settings.

## Quick reference

| Provider | Connection type | Where to enter credentials | Notes |
| --- | --- | --- | --- |
| Google Drive | OAuth | Settings -> Google Drive Credentials | Uses OAuth client ID and secret |
| OneDrive | OAuth | Settings -> OneDrive / MS Graph Credentials | Uses Microsoft Entra app registration |
| Dropbox | OAuth | Settings -> Dropbox Credentials | Uses Dropbox app key and secret |
| Box | OAuth | Settings -> Box Credentials | Uses Box app credentials |
| Yandex Disk | OAuth | Settings -> Yandex Disk Credentials | Uses Yandex OAuth app credentials |
| pCloud | OAuth | Settings -> pCloud Credentials | Uses pCloud OAuth app credentials |
| MEGA | Direct login | Cloud Accounts -> Connect MEGA | Uses account email and password (+ 2FA if enabled) |
| Koofr | Direct login | Cloud Accounts -> Connect Koofr | Uses account username/email and App Password |
| MediaFire | Direct login | Cloud Accounts -> Connect MediaFire | Uses account email and password |
| 4Shared | Direct login | Cloud Accounts -> Connect 4Shared | Uses account email and password |
| Backblaze B2 | Access keys | Cloud Accounts -> Connect Backblaze B2 | Uses Application Key ID, Key, and Bucket Name |
| Windows Share (SMB) | Direct login | Cloud Accounts -> Connect Windows Share (SMB) | Uses Host/IP, Share Name, Username, and Password |
| FTP Server | Direct login | Cloud Accounts -> Connect FTP Server | Uses Host, Port (21), Username, Password, and Base Dir |
| SFTP (SSH) | Direct login | Cloud Accounts -> Connect SFTP (SSH) | Uses Host, Port (22), Username, Password, and Base Dir |
| WebDAV | Direct login | Cloud Accounts -> Connect WebDAV | Uses server URL, username, and password/token |
| S3-compatible storage | Access keys | Cloud Accounts -> Connect S3 | Uses endpoint, bucket, access key, and secret key |
| Telegram Bot | Bot token | Cloud Accounts -> Connect Telegram | Uses bot token and chat ID/channel ID |
| Telegram User | MTProto login | Cloud Accounts -> Connect Telegram User | Uses phone number, API ID, API hash, and verification code |

## What needs OAuth, and what does not

Awd DriveRouter supports two connection styles:

| Provider | OAuth app required? | What you prepare | Where you connect |
| --- | --- | --- | --- |
| Google Drive | Yes (OAuth client in Google Cloud) | Client ID + secret in Settings | Connect -> Google Drive (redirect login) |
| OneDrive | Yes (Entra app registration) | Client ID + secret in Settings | Connect -> OneDrive (redirect login) |
| Dropbox | Yes (Dropbox app) | App key + secret in Settings | Connect -> Dropbox (redirect login) |
| Box | Yes (Box app) | Client ID + secret in Settings | Connect -> Box (redirect login) |
| Yandex Disk | Yes (Yandex OAuth app) | Client ID + secret in Settings | Connect -> Yandex Disk (redirect login) |
| pCloud | Yes (pCloud App Console) | Client ID + secret in Settings | Connect -> pCloud (redirect login) |
| MEGA | No | Email + password (+ 2FA) | Connect -> MEGA (in-app form) |
| WebDAV | No | Server URL + Username + Password | Connect -> WebDAV (in-app form) |
| S3 (R2, B2, MinIO, Tebi, etc.) | No | Access Key + Secret + bucket + endpoint | Connect -> S3 (in-app form) |
| Telegram (Bot & User) | No | Bot token or Phone + API ID/Hash | Connect -> Telegram (in-app form) |

## Google Drive

1. Open Google Cloud Console (`https://console.cloud.google.com/`).
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

1. Open the Microsoft Entra app registration page (`https://entra.microsoft.com/`).
2. Register a new application.
3. Set the supported account type according to your tenant needs.
4. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

5. Add Microsoft Graph delegated permissions for files and profile access (`Files.ReadWrite.All`, `offline_access`, `User.Read`).
6. Create a client secret.
7. Save the client ID and secret in the app settings.
8. Open Cloud Accounts and connect OneDrive.

## Dropbox

1. Open Dropbox App Console (`https://www.dropbox.com/developers/apps`).
2. Create an app with scoped access and full Dropbox or app folder permissions.
3. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

4. Copy the App key and App secret into the Dropbox credentials section in Settings.
5. Enable permissions: `account_info.read`, `files.metadata.read`, `files.content.read`, `files.content.write`.
6. Open Cloud Accounts and connect Dropbox.

## Box

1. Open the Box Developer Console (`https://app.box.com/developers/console`).
2. Create a Custom App with OAuth 2.0 authentication.
3. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

4. Save the Client ID and Client Secret in Settings.
5. Open Cloud Accounts and connect Box.

## Yandex Disk

1. Open the Yandex OAuth app console (`https://oauth.yandex.com/client/new`).
2. Create a new application (Web services platform).
3. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

4. Enable Yandex.Disk permissions (`cloud_api:disk.read`, `cloud_api:disk.write`, `cloud_api:disk.info`).
5. Save the client ID and client secret in Settings.
6. Open Cloud Accounts and connect Yandex Disk.

## MEGA

1. Open Cloud Accounts.
2. Choose Connect -> MEGA.
3. Enter your MEGA email and password.
4. Submit the form.

### Notes

- No Client ID or Client Secret required.
- Session credentials are stored encrypted in the local SQLite database.

## Koofr

1. Open Cloud Accounts.
2. Choose Connect -> Koofr.
3. Enter your Koofr Email/Username and App Password (generated from Koofr Settings -> Account -> Password -> Generate App Password).
4. Submit the form.

### Notes

- Includes 10 GB free storage tier.
- Connected via secure WebDAV protocol (`https://app.koofr.net/dav/Koofr`).

## MediaFire

1. Open Cloud Accounts.
2. Choose Connect -> MediaFire.
3. Enter your MediaFire Email and Password.
4. Submit the form.

### Notes

- Includes 10 GB free storage tier.

## 4Shared

1. Open Cloud Accounts.
2. Choose Connect -> 4Shared.
3. Enter your 4Shared Email and Password.
4. Submit the form.

### Notes

- Includes 15 GB free storage tier.

## Backblaze B2 Native

1. Open Cloud Accounts.
2. Choose Connect -> Backblaze B2.
3. Enter Key ID, Application Key, and Bucket Name.
4. Submit the form.

### Notes

- Includes 10 GB free storage tier.

## Windows Network Share (SMB / LAN)

1. Open Cloud Accounts.
2. Choose Connect -> Windows Share (SMB).
3. Enter Server Host/IP (e.g. `192.168.1.100`), Share Name, Username, and Password.
4. Submit the form.

### Notes

- Native NTLM authentication over SMB2 protocol (`github.com/hirochachacha/go-smb2`).

## FTP Server

1. Open Cloud Accounts.
2. Choose Connect -> FTP Server.
3. Enter Host/IP, Port (default 21), Username, Password, and Base Remote Directory (e.g. `/`).
4. Submit the form.

## SFTP (SSH) Server

1. Open Cloud Accounts.
2. Choose Connect -> SFTP (SSH).
3. Enter Host/IP, Port (default 22), Username, Password, and Base Remote Directory (e.g. `/`).
4. Submit the form.

## pCloud

1. Open the pCloud App Console (`https://docs.pcloud.com/my_apps/`).
2. Click **Create new app**.
3. Add this redirect URI:

   `http://localhost:5998/oauth/callback`

4. Copy the Client ID and Client Secret into the pCloud Credentials section in App Settings.
5. Open Cloud Accounts and connect pCloud.

### Notes

- pCloud uses OAuth 2.0 authentication like Google, OneDrive, Dropbox, Box, and Yandex.
- Ensure Client ID and Client Secret are saved in Settings before connecting.

## WebDAV

1. Open Cloud Accounts.
2. Choose Connect -> WebDAV.
3. Enter server URL, username, and password/token.
4. Submit the form.

## S3-compatible storage

1. Open Cloud Accounts.
2. Choose Connect -> S3.
3. Enter Access Key ID, Secret Access Key, Bucket Name, Endpoint, and Region.
4. Submit the form.

## Telegram (Bot & User)

1. Open Cloud Accounts.
2. Choose Connect -> Telegram (Bot) or Connect -> Telegram User.
3. For Bot: enter Bot Token and Target Chat ID.
4. For User: enter Phone Number, API ID, API Hash, and 2FA password (if enabled).
5. Submit the form.
