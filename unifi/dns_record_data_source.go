package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/svilendotorg/go-unifi-api-integration-v1/unifi"
)

var _ datasource.DataSource = &dnsRecordDataSource{}

func NewDNSRecordDataSource() datasource.DataSource {
	return &dnsRecordDataSource{}
}

type dnsRecordDataSource struct {
	client *Client
}

type dnsRecordDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Site    types.String `tfsdk:"site"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Value   types.String `tfsdk:"value"`
	TTL     types.Int64  `tfsdk:"ttl"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (d *dnsRecordDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (d *dnsRecordDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for DNS records.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this DNS record.",
				Computed:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the DNS record is associated with.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the DNS record to look up.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the DNS record.",
				Computed:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the DNS record.",
				Computed:            true,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "The TTL of the DNS record.",
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the DNS record is enabled.",
				Computed:            true,
			},
		},
	}
}

func (d *dnsRecordDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	d.client = client
}

func (d *dnsRecordDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data dnsRecordDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	name := data.Name.ValueString()

	dnsRecords, err := d.client.ListDNSRecord(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading DNS Records",
			"Could not read DNS records: "+err.Error(),
		)
		return
	}

	var dnsRecord *unifi.DNSRecord
	for _, record := range dnsRecords {
		if (name == "" && record.HiddenID == "default") || record.Domain == name {
			dnsRecord = &record
			break
		}
	}

	if dnsRecord == nil {
		resp.Diagnostics.AddError(
			"DNS Record Not Found",
			fmt.Sprintf("DNS record with name %s not found", name),
		)
		return
	}

	// Map UniFi API type to Terraform record type
	typeMapping := map[string]string{
		"A_RECORD":     "A",
		"AAAA_RECORD":  "AAAA",
		"CNAME_RECORD": "CNAME",
		"MX_RECORD":    "MX",
		"TXT_RECORD":   "TXT",
		"SRV_RECORD":   "SRV",
		"PTR_RECORD":   "PTR",
	}

	data.ID = types.StringValue(dnsRecord.ID)
	data.Site = types.StringValue(site)
	data.Name = types.StringValue(dnsRecord.Domain)
	data.Type = types.StringValue(typeMapping[dnsRecord.Type])
	data.Enabled = types.BoolValue(dnsRecord.Enabled)

	// Set the appropriate value field based on record type
	switch dnsRecord.Type {
	case "A_RECORD":
		data.Value = types.StringValue(dnsRecord.IPv4Address)
	case "AAAA_RECORD":
		data.Value = types.StringValue(dnsRecord.IPv6Address)
	default:
		// For CNAME, MX, PTR, TXT, SRV, use TargetDomain
		data.Value = types.StringValue(dnsRecord.TargetDomain)
	}

	data.TTL = types.Int64Value(dnsRecord.TTLSeconds)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
