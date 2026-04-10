package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &dnsRecordFrameworkResource{}
	_ resource.ResourceWithImportState = &dnsRecordFrameworkResource{}
)

func NewDNSRecordFrameworkResource() resource.Resource {
	return &dnsRecordFrameworkResource{}
}

// dnsRecordFrameworkResource defines the resource implementation.
type dnsRecordFrameworkResource struct {
	client *Client
}

// dnsRecordFrameworkResourceModel describes the resource data model.
type dnsRecordFrameworkResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Site       types.String `tfsdk:"site"`
	Name       types.String `tfsdk:"name"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	Port       types.Int64  `tfsdk:"port"`
	Priority   types.Int64  `tfsdk:"priority"`
	RecordType types.String `tfsdk:"record_type"`
	TTL        types.Int64  `tfsdk:"ttl"`
	Value      types.String `tfsdk:"value"`
	Weight     types.Int64  `tfsdk:"weight"`
}

func (r *dnsRecordFrameworkResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordFrameworkResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages DNS record settings for different providers.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the DNS record.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the DNS record with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The key of the DNS record.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the DNS record is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "The port of the DNS record.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 65535),
				},
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "The priority of the DNS record.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"record_type": schema.StringAttribute{
				MarkdownDescription: "The type of the DNS record.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("A", "AAAA", "CNAME", "MX", "TXT", "SRV", "PTR"),
				},
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "The TTL of the DNS record.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtMost(65535),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the DNS record.",
				Required:            true,
			},
			"weight": schema.Int64Attribute{
				MarkdownDescription: "The weight of the DNS record.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
		},
	}
}

func (r *dnsRecordFrameworkResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	r.client = client
}

func (r *dnsRecordFrameworkResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data dnsRecordFrameworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to unifi.DNSRecord
	dnsRecord := r.modelToDNSRecord(ctx, &data)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Create the DNS record
	createdDNSRecord, err := r.client.CreateDNSRecord(ctx, site, dnsRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Dns Record",
			err.Error(),
		)
		return
	}

	// Convert back to model
	r.dnsRecordToModel(ctx, createdDNSRecord, &data, site)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsRecordFrameworkResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data dnsRecordFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Get the DNS record from the API
	dnsRecord, err := r.client.GetDNSRecord(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Dns Record",
			"Could not read DNS record with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Convert to model
	r.dnsRecordToModel(ctx, dnsRecord, &data, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsRecordFrameworkResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state dnsRecordFrameworkResourceModel
	var plan dnsRecordFrameworkResourceModel

	// Step 1: Read the current state (which already contains API values from previous reads)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 2: Apply the plan changes to the state object
	r.applyPlanToState(ctx, &plan, &state)

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Step 3: Convert the updated state to API format
	dnsRecord := r.modelToDNSRecord(ctx, &state)
	dnsRecord.ID = state.ID.ValueString()

	// Step 4: Send to API
	updatedDNSRecord, err := r.client.UpdateDNSRecord(ctx, site, dnsRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Dns Record",
			err.Error(),
		)
		return
	}

	// Step 5: Update state with API response
	r.dnsRecordToModel(ctx, updatedDNSRecord, &state, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRecordFrameworkResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data dnsRecordFrameworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Delete the DNS record
	err := r.client.DeleteDNSRecord(ctx, site, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Dns Record",
			err.Error(),
		)
		return
	}
}

func (r *dnsRecordFrameworkResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Import format: "site:id" or just "id" for default site
	idParts := strings.Split(req.ID, ":")

	if len(idParts) == 2 {
		// site:id format
		site := idParts[0]
		id := idParts[1]

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	if len(idParts) == 1 {
		// Just id, use default site
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
		return
	}

	resp.Diagnostics.AddError(
		"Invalid Import ID",
		"Import ID must be in format 'site:id' or 'id'",
	)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *dnsRecordFrameworkResource) applyPlanToState(
	_ context.Context,
	plan *dnsRecordFrameworkResourceModel,
	state *dnsRecordFrameworkResourceModel,
) {
	// Apply plan values to state, but only if plan value is not null/unknown
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
	}
	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		state.Port = plan.Port
	}
	if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
		state.Priority = plan.Priority
	}
	if !plan.RecordType.IsNull() && !plan.RecordType.IsUnknown() {
		state.RecordType = plan.RecordType
	}
	if !plan.TTL.IsNull() && !plan.TTL.IsUnknown() {
		state.TTL = plan.TTL
	}
	if !plan.Value.IsNull() && !plan.Value.IsUnknown() {
		state.Value = plan.Value
	}
	if !plan.Weight.IsNull() && !plan.Weight.IsUnknown() {
		state.Weight = plan.Weight
	}
}

// modelToDNSRecord converts the Terraform model to the API struct.
func (r *dnsRecordFrameworkResource) modelToDNSRecord(
	_ context.Context,
	model *dnsRecordFrameworkResourceModel,
) *unifi.DNSRecord {
	recordType := model.RecordType.ValueString()
	if recordType == "" {
		recordType = "A"
	}

	// Map Terraform record type to UniFi API type
	typeMapping := map[string]string{
		"A":     "A_RECORD",
		"AAAA":  "AAAA_RECORD",
		"CNAME": "CNAME_RECORD",
		"MX":    "MX_RECORD",
		"TXT":   "TXT_RECORD",
		"SRV":   "SRV_RECORD",
		"PTR":   "PTR_RECORD",
	}

	dnsRecord := &unifi.DNSRecord{
		Domain:     model.Name.ValueString(),
		Enabled:    model.Enabled.ValueBool(),
		Type:       typeMapping[recordType],
		TTLSeconds: model.TTL.ValueInt64(),
	}

	// Set the appropriate address field based on record type
	switch recordType {
	case "A":
		dnsRecord.IPv4Address = model.Value.ValueString()
	case "AAAA":
		dnsRecord.IPv6Address = model.Value.ValueString()
	case "CNAME", "MX", "PTR":
		dnsRecord.TargetDomain = model.Value.ValueString()
	case "TXT", "SRV":
		// For TXT and SRV, use TargetDomain as a fallback for the value
		dnsRecord.TargetDomain = model.Value.ValueString()
	}

	if !model.Enabled.IsNull() {
		dnsRecord.Enabled = model.Enabled.ValueBool()
	}

	if !model.TTL.IsNull() {
		dnsRecord.TTLSeconds = model.TTL.ValueInt64()
	}

	return dnsRecord
}

// dnsRecordToModel converts the API struct to the Terraform model.
func (r *dnsRecordFrameworkResource) dnsRecordToModel(
	_ context.Context,
	dnsRecord *unifi.DNSRecord,
	model *dnsRecordFrameworkResourceModel,
	site string,
) {
	model.ID = types.StringValue(dnsRecord.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringValue(dnsRecord.Domain)
	model.Enabled = types.BoolValue(dnsRecord.Enabled)

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

	if dnsRecord.Type != "" {
		model.RecordType = types.StringValue(typeMapping[dnsRecord.Type])
	} else {
		model.RecordType = types.StringNull()
	}

	// Set the appropriate value field based on record type
	switch dnsRecord.Type {
	case "A_RECORD":
		model.Value = types.StringValue(dnsRecord.IPv4Address)
	case "AAAA_RECORD":
		model.Value = types.StringValue(dnsRecord.IPv6Address)
	default:
		// For CNAME, MX, PTR, TXT, SRV, use TargetDomain
		model.Value = types.StringValue(dnsRecord.TargetDomain)
	}

	if dnsRecord.TTLSeconds != 0 {
		model.TTL = types.Int64Value(dnsRecord.TTLSeconds)
	} else {
		model.TTL = types.Int64Null()
	}

	// Port, Priority, and Weight are not supported in the new API
	model.Port = types.Int64Null()
	model.Priority = types.Int64Null()
	model.Weight = types.Int64Null()
}
