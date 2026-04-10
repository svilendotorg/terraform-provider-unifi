# terraform-provider-unifi

**Fork of [ubiquiti-community/terraform-provider-unifi](https://github.com/ubiquiti-community/terraform-provider-unifi) with fixes for UniFi Network API 9.x+**

[![GoDoc](https://godoc.org/github.com/svilendotorg/terraform-provider-unifi?status.svg)](https://godoc.org/github.com/svilendotorg/terraform-provider-unifi)

Terraform provider for managing UniFi Network devices. This fork includes critical fixes for UniFi Network API 9.x+ compatibility.

## ⚠️ Important

You can't configure your network while connected to something that may disconnect (like WiFi). Use a hard-wired connection to your controller when using this provider.

## Key Differences from Original

### Fixed Resources (Network API 9.x+)

| Resource | Original Endpoint | Fixed Endpoint | Version |
|----------|-------------------|----------------|---------|
| DNS Records | `v2/api/site/{site}/static-dns` (404) | `integration/v1/sites/{site}/dns/policies` | v0.41.26+ |
| Firewall Policies | `v2/api/site/{site}/firewall-policies` | `integration/v1/sites/{site}/firewall/policies` | v0.41.28+ |

### Why This Fork?

The original provider uses deprecated API endpoints that return 404 errors on UniFi Network API 9.x+. This fork updates to the new `integration/v1` API endpoints.

## Quick Start

### Terraform

```hcl
terraform {
  required_providers {
    unifi = {
      source  = "svilendotorg/unifi"
      version = "0.41.28"
    }
  }
}

provider "unifi" {
  api_url        = "https://192.168.1.1" # enter your UniFi controller IP
  api_key        = "your_api_key"        # see below for how to generate
  allow_insecure = true                  # set to false if you have valid SSL cert
}

# DNS Record
resource "unifi_dns_record" "example" {
  name        = "example.something.lan"
  value       = "192.168.1.100"
  record_type = "A"
  ttl         = 300
  site        = "default"  # or use site UUID
}

# Firewall Policy
resource "unifi_firewall_policy" "allow-internal" {
  name        = "Allow Internal Traffic"
  enabled     = true
  site        = "your_site_uuid"  # site UUID required for firewall policies
  
  action = {
    type                 = "ALLOW"
    allow_return_traffic = true
  }
  
  source = {
    zone_id = "zone_uuid"  # Internal zone
  }
  
  destination = {
    zone_id = "zone_uuid"  # Internal zone
  }
  
  ip_protocol_scope = {
    ip_version = "IPV4"
  }
  
  logging_enabled = false
}
```

### Configuration Details

#### API Key Generation

1. Navigate to: `https://<unifi-controller-ip>/network/default/integrations`
2. Click **"Create New Key"** or **"Add API Key"**
3. Give it a name (e.g., "Terraform")
4. Select **"Full Access"** or **"System Administrator"** permissions
5. Click **"Create"** and copy the API key immediately (you won't see it again)
6. Store it securely in your Terraform variables or environment

**Important**: For firewall policies, the API key must have **Full Access** permissions. Limited keys will work for DNS records but fail for firewall operations.

#### Site Parameter

The `site` parameter can be either:
- **Site name**: `"default"` (case-sensitive, matches the site name in UniFi UI)
- **Site UUID**: `"xxxxxxxx-xxx-xxx-xxx-xxxxxxxxxxxx"` (more reliable)

**Finding your site UUID**:
```bash
curl -k -s "https://<controller>/proxy/network/integration/v1/sites" \
  -H "X-API-KEY: your_api_key" | jq '.data[] | {name, id}'
```

**Recommendation**: Use site UUIDs for firewall policies, site names work fine for DNS records.

**Finding your zone IDs**:
```bash
curl -k -s "https://<controller>/proxy/network/integration/v1/sites/<site-id>/firewall/zones" \
  -H "X-API-KEY: your_api_key" | jq '.data[] | {name, id}'
```

**Note**: Zone UUIDs are typically the same across all UniFi controllers, but verify with the command above.

### OpenTofu

This provider also works with OpenTofu:

```hcl
terraform {
  required_providers {
    unifi = {
      source  = "svilendotorg/unifi"
      version = "0.41.28"
    }
  }
}
```

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
| v0.41.28 | Firewall policy integration/v1 API support |
| v0.41.27 | Bug fixes and updates |
| v0.41.26 | DNS record integration/v1 API support |
| v0.41.25 | Base fork version |

## Documentation

- **Provider Docs**: https://registry.terraform.io/providers/svilendotorg/unifi/latest/docs
- **API Documentation**: [[unifi-api-management]] (local Obsidian vault)
- **Go SDK**: https://github.com/svilendotorg/go-unifi-api-integration-v1

## Development

This provider uses the [go-unifi-api-integration-v1](https://github.com/svilendotorg/go-unifi-api-integration-v1) SDK fork.

Functionality first needs to be added to the go-unifi SDK before it can be used in the provider.

## License

Mozilla Public License 2.0 (same as original terraform-provider-unifi)
