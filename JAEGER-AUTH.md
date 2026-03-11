# Jaeger Authentication Guide

The Jaeger tracing UI uses Google Cloud Identity-Aware Proxy (IAP) for authentication.

## Quick Copy-Paste

You need **both** cookies in this format:

```
GCP_IAP_UID=<your-uid>; __Host-GCP_IAP_AUTH_TOKEN_=<your-token>
```

## How to Get Your Cookies

### Method 1: Browser DevTools (Recommended)

1. Open https://tools.masstack.com/tracing in your browser
2. Press `F12` to open DevTools
3. Go to **Application** tab (Chrome) or **Storage** tab (Firefox)
4. Click **Cookies** → `https://tools.masstack.com`
5. Find these two cookies:
   - `GCP_IAP_UID`
   - `__Host-GCP_IAP_AUTH_TOKEN_`
6. Copy both values

### Method 2: Network Tab

1. Open https://tools.masstack.com/tracing
2. Press `F12` → **Network** tab
3. Refresh the page
4. Click any request to the API
5. Look at **Request Headers**
6. Find the `Cookie:` header
7. Copy the entire value

## Example

If you see these cookies in your browser:
- `GCP_IAP_UID` = `117869924448184822733`
- `__Host-GCP_IAP_AUTH_TOKEN_` = `ATJDtjRMPPwJU3FwKkY32GKx2UXixZOkmAdNZ78CYLOKKaLietdv5xID5vW7eEVfRorz8jub7vi5vedhZ`

Paste this into the TUI:
```
GCP_IAP_UID=117869924448184822733; __Host-GCP_IAP_AUTH_TOKEN_=ATJDtjRMPPwJU3FwKkY32GKx2UXixZOkmAdNZ78CYLOKKaLietdv5xID5vW7eEVfRorz8jub7vi5vedhZ
```

## Notes

- Both cookies are required for authentication
- The semicolon (`;`) between them is important
- Cookies expire after some time - if you get auth errors, refresh your cookies
- You must be on VPN to access the service

## Troubleshooting

**"401 Unauthorized" or "403 Forbidden"**
- Your cookies may have expired
- Get fresh cookies from the browser
- Make sure you're connected to VPN

**"No services found"**
- Authentication succeeded but no data
- Check if Jaeger has any traces in the UI
- Try selecting a different time range (the tool uses last 24 hours)
