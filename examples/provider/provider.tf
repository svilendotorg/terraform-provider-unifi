provider "unifi" {
  username = var.username # optionally use UNIFI_USERNAME env var
  password = var.password # optionally use UNIFI_PASSWORD env var
  api_url  = var.api_url  # optionally use UNIFI_API env var
  api_key  = var.api_key  # optionally use UNIFI_API_KEY

  # you may need to allow insecure TLS communications unless you have configured
  # certificates for your controller
  allow_insecure = var.insecure # optionally use UNIFI_INSECURE env var

  # For integration/v1 API resources (DNS records, firewall policies), use site UUID
  # site = "<your-site-uuid>" or optionally use UNIFI_SITE env var
  # To find your site UUID:
  # curl -k -s "https://<controller>/proxy/network/integration/v1/sites" \
  #   -H "X-API-KEY: <your-api-key>" | jq '.data[] | {name, id}'
}
