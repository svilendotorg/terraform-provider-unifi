# terraform-provider-unifi

**Fork of [ubiquiti-community/terraform-provider-unifi](https://github.com/ubiquiti-community/terraform-provider-unifi) with fixes for UniFi Network API 9.x+**

[![GoDoc](https://godoc.org/github.com/svilendotorg/terraform-provider-unifi?status.svg)](https://godoc.org/github.com/svilendotorg/terraform-provider-unifi)

Terraform provider for managing UniFi Network devices. This fork includes critical fixes for UniFi Network API 9.x+ compatibility.

## ⚠️ Important

You can't configure your network while connected to something that may disconnect (like WiFi). Use a hard-wired connection to your controller when using this provider.

> **Note**: This project is developed and tested for **direct connection** mode only. Cloud Connector mode is not tested and its functionality is unknown.

> **Note**: Two-factor authentication (2FA) is **not supported** by this provider. Use API keys for authentication.

## Key Differences from Original

### Fixed Resources (Network API 9.x+)

| Resource | Original Endpoint | Fixed Endpoint | Version |
|----------|-------------------|----------------|---------|
| DNS Records | `v2/api/site/{site}/static-dns` (404) | `integration/v1/sites/{site}/dns/policies` | v0.41.26+ |
| Firewall Policies | `v2/api/site/{site}/firewall-policies` | `integration/v1/sites/{site}/firewall/policies` | v0.41.28+ |
| Firewall Policies (full CRUD) | - | JSON parsing fixes | v0.41.29+ |

### Why This Fork?

The original provider uses deprecated API endpoints that return 404 errors on UniFi Network API 9.x+. This fork updates to the new `integration/v1` API endpoints.

## Quick Start

### UniFi Setup

1. **Create Super Admin User**: It's recommended to create a new local **Super Admin** user at `https://<controller>/network/default/admins/users` (lower permission users don't work)
2. **Create API Key**: Login with the user and create a new API key at `https://<controller>/network/default/integrations/api-key/new` with **Full Access** permissions
3. **Find Site UUID**: Run the following to get your site UUID:
   ```bash
   curl -k -s "https://<controller>/proxy/network/integration/v1/sites" \
     -H "X-API-KEY: <your-api-key>" | jq '.data[] | {name, id}'
   ```

### Terraform / OpenTofu

This provider works with both Terraform and OpenTofu:

```hcl
terraform {
  required_providers {
    unifi = {
      source  = "svilendotorg/unifi"
      version = "0.41.29"
    }
  }
}

provider "unifi" {
  api_url        = "https://192.168.1.1"  # enter your UniFi controller IP
  api_key        = "your_api_key"         # see below for how to generate
  allow_insecure = true                   # set to false if you have valid SSL cert
  site           = "<your-site-uuid>"     # use site UUID for integration/v1 API
}

# DNS Record (works with both Terraform and OpenTofu)
resource "unifi_dns_record" "example" {
  name        = "example.something.lan"
  value       = "192.168.1.100"
  record_type = "A"
  ttl         = 300
  enabled     = true  # required by integration/v1 API
}

# Firewall Policy (Zone-to-Zone)
resource "unifi_firewall_policy" "allow-internal" {
  name                = "Allow Internal Traffic"
  enabled             = true
  action              = "ALLOW"
  allow_return_traffic = true
  source_zone_id      = "<internal-zone-uuid>"  # Replace with your Internal zone UUID
  destination_zone_id = "<internal-zone-uuid>"  # Replace with your Internal zone UUID
  ip_version          = "IPV4"
  logging_enabled     = false
}

# Firewall Policy (IP-to-IP)
resource "unifi_firewall_policy" "allow-specific" {
  name                = "Allow Specific IPs"
  enabled             = true
  action              = "ALLOW"
  allow_return_traffic = true
  source_zone_id      = "<internal-zone-uuid>"
  source_ips          = ["192.168.1.100", "192.168.1.101"]
  destination_zone_id = "<internal-zone-uuid>"
  destination_ips     = ["192.168.1.200"]
  ip_version          = "IPV4"
  logging_enabled     = true
}
```

### API Key Generation

1. Navigate to: `https://<unifi-controller-ip>/network/default/integrations`
2. Click **"Create New Api Key"** - *The permissions of the user will be inherited for the API Key*
3. Set Name, Description and Expiry
4. Click **"Create"** and copy the API key immediately (you won't see it again)
5. Store it securely in your Terraform variables or environment

**Important**: For firewall policies, the API key must have **Full Access** permissions. Limited keys will work for DNS records but fail for firewall operations.

### Finding Site UUID and Zone IDs

The `site` parameter in the provider config should use the **site UUID** for integration/v1 API resources (DNS records, firewall policies).

**Commands to get UUIDs**:
```bash
# Get site UUID
curl -k -s "https://<controller>/proxy/network/integration/v1/sites" \
  -H "X-API-KEY: <your-api-key>" | jq '.data[] | {name, id}'

# Get zone IDs (replace <site-uuid> with your site UUID)
curl -k -s "https://<controller>/proxy/network/integration/v1/sites/<site-uuid>/firewall/zones" \
  -H "X-API-KEY: <your-api-key>" | jq '.data[] | {name, id}'
```

Always use the UUIDs from your specific controller.

## Registry Locations

- **Terraform Registry**: https://registry.terraform.io/providers/svilendotorg/unifi/latest
- **OpenTofu Registry**: https://search.opentofu.org/provider/svilendotorg/unifi/latest

## Supported UniFi Controller Versions

- **UniFi Network API**: 9.x, 10.x, 10.2.105+
- **UniFi OS**: 5.x+

Tested on UCG Ultra.

## Version History

| Version | Changes |
|---------|---------|
| v0.41.29 | **Firewall policy full CRUD support** - Fixed JSON parsing (ID field, list wrapper, ConnectionStateFilter) |
| v0.41.28 | Firewall policy integration/v1 API support (create only, no ID capture) |
| v0.41.27 | Bug fixes and updates |
| v0.41.26 | DNS record integration/v1 API support |
| v0.41.25 | Base fork version |

### Known Issues (v0.41.28 and earlier)

- **Firewall policy delete fails with 405**: The ID was not captured after creation, causing delete to fail
- **Fix**: Use v0.41.29+ with SDK v1.33.51+ for full CRUD support

## Documentation

- **Provider Docs**: https://registry.terraform.io/providers/svilendotorg/unifi/latest/docs

## Development

- **Go SDK** : this provider uses the [go-unifi-api-integration-v1](https://github.com/svilendotorg/go-unifi-api-integration-v1) SDK fork, containing fixes for UniFi Network API

- Functionality must be added to the go-unifi SDK before it can be used in the provider.

## License

Mozilla Public License 2.0 (same as original terraform-provider-unifi)
