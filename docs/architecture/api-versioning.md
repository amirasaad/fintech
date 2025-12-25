---
icon: material/link
---

# 🧩 API Versioning Strategy

This document outlines the API versioning strategy for the FinTech API to ensure backward compatibility and smooth transitions between versions.

## 📌 Versioning Scheme

- **Current Version**: 1.1
- **Version Format**: `v{major}.{minor}` (e.g., `v1.1`)
- **Default Version**: `v1.0` (when no version is specified)

## 📌 Version Identification

API versions can be specified using one of the following methods (in order of precedence):

1. **URL Path** (Recommended):

   ```http
   GET /v1.1/accounts
   ```

2. **Accept Header**:

   ```http
   Accept: application/vnd.fintech.v1.1+json
   ```

3. **Custom Header**:

   ```http
   X-API-Version: 1.1
   ```

## 📌 Backward Compatibility Policy

### Version 1.1 (Current)

- **Changes**:
  - Added support for transaction fees
  - Added new exchange rate caching mechanism
- **Backward Compatibility**: Fully backward compatible with v1.0

### Version 1.0 (Legacy)

- Initial stable release
- No breaking changes from previous versions

## 📌 Handling Breaking Changes

For future breaking changes, we will:

1. Increment the major version number
2. Maintain the previous major version for a reasonable deprecation period
3. Provide migration guides and tools when possible
4. Clearly document breaking changes in release notes

## 📌 Best Practices for Clients

1. Always specify the API version in requests
2. Handle HTTP 400 responses for unsupported versions
3. Test new versions in a staging environment before upgrading production
4. Monitor deprecation notices for the APIs you use

## 📌 Example Requests

### Using URL Path

```http
GET /v1.1/accounts/123/transactions HTTP/1.1
Host: api.fintech.example.com
```

### Using Accept Header

```http
GET /accounts/123/transactions HTTP/1.1
Host: api.fintech.example.com
Accept: application/vnd.fintech.v1.1+json
```

## 📌 Version Discovery

You can discover available versions by making a request to the root endpoint:

```http
GET / HTTP/1.1
Host: api.fintech.example.com
```

Response:

```json
{
  "versions": ["1.0", "1.1"],
  "current_version": "1.1",
  "min_supported_version": "1.0"
}
```
