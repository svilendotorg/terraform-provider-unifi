package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/svilendotorg/go-unifi-api-integration-v1/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &firewallPolicyResource{}
	_ resource.ResourceWithImportState = &firewallPolicyResource{}
)

func NewFirewallPolicyFrameworkResource() resource.Resource {
	return &firewallPolicyResource{}
}

// firewallPolicyResource defines the resource implementation.
type firewallPolicyResource struct {
	client *Client
}

// firewallPolicyResourceModel describes the resource data model.
type firewallPolicyResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Site               types.String `tfsdk:"site"`
	Name               types.String `tfsdk:"name"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	Description        types.String `tfsdk:"description"`
	Action             types.String `tfsdk:"action"`
	AllowReturnTraffic types.Bool   `tfsdk:"allow_return_traffic"`
	SourceZoneID       types.String `tfsdk:"source_zone_id"`
	SourceIPs          types.Set    `tfsdk:"source_ips"`
	DestZoneID         types.String `tfsdk:"destination_zone_id"`
	DestIPs            types.Set    `tfsdk:"destination_ips"`
	IPVersion          types.String `tfsdk:"ip_version"`
	LoggingEnabled     types.Bool   `tfsdk:"logging_enabled"`
}

func (r *firewallPolicyResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_firewall_policy"
}

func (r *firewallPolicyResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "`unifi_firewall_policy` manages firewall policies using the integration/v1 API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the firewall policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				Description: "The site UUID to associate the firewall policy with.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the firewall policy.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the firewall policy is enabled.",
				Computed:    true,
				Optional:    true,
				Default:     booldefault.StaticBool(true),
			},
			"description": schema.StringAttribute{
				Description: "Optional description of the firewall policy.",
				Optional:    true,
			},
			"action": schema.StringAttribute{
				Description: "The action to take: ALLOW, BLOCK, or REJECT.",
				Required:    true,
			},
			"allow_return_traffic": schema.BoolAttribute{
				Description: "Whether to allow return traffic (only for ALLOW action).",
				Computed:    true,
				Optional:    true,
				Default:     booldefault.StaticBool(true),
			},
			"source_zone_id": schema.StringAttribute{
				Description: "The source zone UUID (e.g., '569b7b10-fb2c-4499-a3d0-67571b77dbe2' for Internal).",
				Required:    true,
			},
			"source_ips": schema.SetAttribute{
				Description: "List of source IP addresses to match.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"destination_zone_id": schema.StringAttribute{
				Description: "The destination zone UUID.",
				Required:    true,
			},
			"destination_ips": schema.SetAttribute{
				Description: "List of destination IP addresses to match.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"ip_version": schema.StringAttribute{
				Description: "IP version: IPV4, IPV6, or BOTH.",
				Computed:    true,
				Optional:    true,
				Default:     stringdefault.StaticString("IPV4"),
			},
			"logging_enabled": schema.BoolAttribute{
				Description: "Whether to log matching traffic.",
				Computed:    true,
				Optional:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *firewallPolicyResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*Client)
}

func (r *firewallPolicyResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data firewallPolicyResourceModel

	diag := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Build source filter
	var sourceIPs []string
	diag = data.SourceIPs.ElementsAs(ctx, &sourceIPs, false)
	resp.Diagnostics.Append(diag...)

	var sourceFilter *unifi.FirewallPolicyTrafficFilter
	if len(sourceIPs) > 0 {
		var items []unifi.FirewallPolicyIPAddressFilterItem
		for _, ip := range sourceIPs {
			items = append(items, unifi.FirewallPolicyIPAddressFilterItem{
				Type:  "IP_ADDRESS",
				Value: ip,
			})
		}
		sourceFilter = &unifi.FirewallPolicyTrafficFilter{
			Type: "IP_ADDRESS",
			IPAddressFilter: &unifi.FirewallPolicyIPAddressFilter{
				Type:          "IP_ADDRESSES",
				MatchOpposite: false,
				Items:         items,
			},
		}
	}

	// Build destination filter
	var destIPs []string
	diag = data.DestIPs.ElementsAs(ctx, &destIPs, false)
	resp.Diagnostics.Append(diag...)

	var destFilter *unifi.FirewallPolicyTrafficFilter
	if len(destIPs) > 0 {
		var items []unifi.FirewallPolicyIPAddressFilterItem
		for _, ip := range destIPs {
			items = append(items, unifi.FirewallPolicyIPAddressFilterItem{
				Type:  "IP_ADDRESS",
				Value: ip,
			})
		}
		destFilter = &unifi.FirewallPolicyTrafficFilter{
			Type: "IP_ADDRESS",
			IPAddressFilter: &unifi.FirewallPolicyIPAddressFilter{
				Type:          "IP_ADDRESSES",
				MatchOpposite: false,
				Items:         items,
			},
		}
	}

	policy := &unifi.FirewallPolicy{
		SiteID:  site,
		Name:    data.Name.ValueString(),
		Enabled: data.Enabled.ValueBool(),
		Action: &unifi.FirewallPolicyAction{
			Type:               data.Action.ValueString(),
			AllowReturnTraffic: data.AllowReturnTraffic.ValueBool(),
		},
		Source: &unifi.FirewallPolicySource{
			ZoneID:        data.SourceZoneID.ValueString(),
			TrafficFilter: sourceFilter,
		},
		Destination: &unifi.FirewallPolicyDestination{
			ZoneID:        data.DestZoneID.ValueString(),
			TrafficFilter: destFilter,
		},
		IPProtocolScope: &unifi.FirewallPolicyIPProtocolScope{
			IPVersion: data.IPVersion.ValueString(),
		},
		LoggingEnabled: data.LoggingEnabled.ValueBool(),
	}

	if !data.Description.IsNull() {
		policy.Description = data.Description.ValueString()
	}

	_, err := r.client.ApiClient.CreateFirewallPolicy(ctx, site, policy)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create firewall policy",
			fmt.Sprintf("Could not create firewall policy: %s", err),
		)
		return
	}

	// API doesn't return ID on create, fetch it by listing policies
	var policies []unifi.FirewallPolicy
	policies, err = r.client.ApiClient.ListFirewallPolicy(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch firewall policy ID",
			fmt.Sprintf("Could not list firewall policies after creation: %s", err),
		)
		return
	}

	// Find the policy by name
	var found *unifi.FirewallPolicy
	for _, p := range policies {
		if p.Name == data.Name.ValueString() && p.ID != "" {
			found = &p
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			"Firewall policy created but not found",
			"Policy was created but could not be retrieved to get its ID",
		)
		return
	}

	data.ID = types.StringValue(found.ID)
	data.Site = types.StringValue(site)

	diag = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diag...)
}

func (r *firewallPolicyResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data firewallPolicyResourceModel

	diag := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, err := r.client.ApiClient.ListFirewallPolicy(ctx, data.Site.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read firewall policies",
			fmt.Sprintf("Could not read firewall policies: %s", err),
		)
		return
	}

	var found *unifi.FirewallPolicy
	for _, p := range policies {
		if p.ID == data.ID.ValueString() {
			found = &p
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(found.ID)
	data.Name = types.StringValue(found.Name)
	data.Enabled = types.BoolValue(found.Enabled)
	data.LoggingEnabled = types.BoolValue(found.LoggingEnabled)

	if found.Description != "" {
		data.Description = types.StringValue(found.Description)
	}

	if found.Action != nil {
		data.Action = types.StringValue(found.Action.Type)
		data.AllowReturnTraffic = types.BoolValue(found.Action.AllowReturnTraffic)
	}

	if found.Source != nil {
		data.SourceZoneID = types.StringValue(found.Source.ZoneID)
	}

	if found.Destination != nil {
		data.DestZoneID = types.StringValue(found.Destination.ZoneID)
	}

	if found.IPProtocolScope != nil {
		data.IPVersion = types.StringValue(found.IPProtocolScope.IPVersion)
	}

	diag = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diag...)
}

func (r *firewallPolicyResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data firewallPolicyResourceModel

	diag := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()

	// Build the policy update (similar to Create)
	var sourceIPs []string
	diag = data.SourceIPs.ElementsAs(ctx, &sourceIPs, false)
	resp.Diagnostics.Append(diag...)

	var sourceFilter *unifi.FirewallPolicyTrafficFilter
	if len(sourceIPs) > 0 {
		var items []unifi.FirewallPolicyIPAddressFilterItem
		for _, ip := range sourceIPs {
			items = append(items, unifi.FirewallPolicyIPAddressFilterItem{
				Type:  "IP_ADDRESS",
				Value: ip,
			})
		}
		sourceFilter = &unifi.FirewallPolicyTrafficFilter{
			Type: "IP_ADDRESS",
			IPAddressFilter: &unifi.FirewallPolicyIPAddressFilter{
				Type:          "IP_ADDRESSES",
				MatchOpposite: false,
				Items:         items,
			},
		}
	}

	var destIPs []string
	diag = data.DestIPs.ElementsAs(ctx, &destIPs, false)
	resp.Diagnostics.Append(diag...)

	var destFilter *unifi.FirewallPolicyTrafficFilter
	if len(destIPs) > 0 {
		var items []unifi.FirewallPolicyIPAddressFilterItem
		for _, ip := range destIPs {
			items = append(items, unifi.FirewallPolicyIPAddressFilterItem{
				Type:  "IP_ADDRESS",
				Value: ip,
			})
		}
		destFilter = &unifi.FirewallPolicyTrafficFilter{
			Type: "IP_ADDRESS",
			IPAddressFilter: &unifi.FirewallPolicyIPAddressFilter{
				Type:          "IP_ADDRESSES",
				MatchOpposite: false,
				Items:         items,
			},
		}
	}

	policy := &unifi.FirewallPolicy{
		ID:      data.ID.ValueString(),
		SiteID:  site,
		Name:    data.Name.ValueString(),
		Enabled: data.Enabled.ValueBool(),
		Action: &unifi.FirewallPolicyAction{
			Type:               data.Action.ValueString(),
			AllowReturnTraffic: data.AllowReturnTraffic.ValueBool(),
		},
		Source: &unifi.FirewallPolicySource{
			ZoneID:        data.SourceZoneID.ValueString(),
			TrafficFilter: sourceFilter,
		},
		Destination: &unifi.FirewallPolicyDestination{
			ZoneID:        data.DestZoneID.ValueString(),
			TrafficFilter: destFilter,
		},
		IPProtocolScope: &unifi.FirewallPolicyIPProtocolScope{
			IPVersion: data.IPVersion.ValueString(),
		},
		LoggingEnabled: data.LoggingEnabled.ValueBool(),
	}

	if !data.Description.IsNull() {
		policy.Description = data.Description.ValueString()
	}

	_, err := r.client.ApiClient.UpdateFirewallPolicy(ctx, site, policy)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update firewall policy",
			fmt.Sprintf("Could not update firewall policy: %s", err),
		)
		return
	}

	diag = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diag...)
}

func (r *firewallPolicyResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data firewallPolicyResourceModel

	diag := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.ApiClient.DeleteFirewallPolicy(ctx, data.Site.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete firewall policy",
			fmt.Sprintf("Could not delete firewall policy: %s", err),
		)
		return
	}
}

func (r *firewallPolicyResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	parts := strings.Split(req.ID, ",")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Error importing firewall policy",
			"Import ID must be in the format: site_id,policy_id",
		)
		return
	}

	siteID := parts[0]
	policyID := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), policyID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), siteID)...)
}
