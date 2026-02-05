package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/revosai/terraform-provider-revos/internal/client"
)

// Ensure implementation satisfies interfaces.
var _ resource.Resource = &OverlayResource{}
var _ resource.ResourceWithImportState = &OverlayResource{}

// jsonSemanticEqualModifier is a plan modifier that suppresses diffs for JSON strings
// that are semantically equal (same content, different key ordering)
type jsonSemanticEqualModifier struct{}

func (m jsonSemanticEqualModifier) Description(ctx context.Context) string {
	return "Suppresses diffs for JSON strings that are semantically equal"
}

func (m jsonSemanticEqualModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m jsonSemanticEqualModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If state is null (new resource), nothing to compare
	if req.StateValue.IsNull() {
		return
	}

	// If config is null or unknown, let terraform handle it
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Compare semantically
	if jsonEqual(req.StateValue.ValueString(), req.ConfigValue.ValueString()) {
		// They're semantically equal, use state value to suppress diff
		resp.PlanValue = req.StateValue
	}
}

// Implement ResourceWithModifyPlan to handle computed field drift
var _ resource.ResourceWithModifyPlan = &OverlayResource{}

func (r *OverlayResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If destroying, nothing to do
	if req.Plan.Raw.IsNull() {
		return
	}

	// If creating, nothing to do
	if req.State.Raw.IsNull() {
		return
	}

	var plan, state OverlayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if name, description, and data are unchanged
	nameUnchanged := plan.Name.Equal(state.Name)
	// Treat null and empty string as equal for description
	descUnchanged := stringEqualOrBothEmpty(plan.Description, state.Description)
	dataUnchanged := jsonEqual(plan.Data.ValueString(), state.Data.ValueString())

	// If all user-controlled fields are unchanged, preserve computed fields from state
	if nameUnchanged && descUnchanged && dataUnchanged {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("organization_id"), state.OrganizationID)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("created_by"), state.CreatedBy)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("created_at"), state.CreatedAt)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("updated_at"), state.UpdatedAt)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("data"), state.Data)...)
	}
}

func NewOverlayResource() resource.Resource {
	return &OverlayResource{}
}

type OverlayResource struct {
	client *client.Client
}

type OverlayResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Data           types.String `tfsdk:"data"` // JSON String
	CreatedBy      types.String `tfsdk:"created_by"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *OverlayResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_overlay"
}

func (r *OverlayResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Revos Cube Overlay.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the overlay. Must be unique.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the overlay.",
			},
			"organization_id": schema.StringAttribute{
				Computed: true,
			},
			"data": schema.StringAttribute{
				Required:      true,
				Description:   "The JSON string representation of the Cube definition.",
				PlanModifiers: []planmodifier.String{jsonSemanticEqualModifier{}},
			},
			"created_by": schema.StringAttribute{
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *OverlayResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *OverlayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OverlayResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var rawData json.RawMessage
	if err := json.Unmarshal([]byte(data.Data.ValueString()), &rawData); err != nil {
		resp.Diagnostics.AddError("Invalid JSON in data", err.Error())
		return
	}

	payload := client.OverlayPayload{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Data:        rawData,
	}

	overlay, err := r.client.CreateOverlay(payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create overlay, got error: %s", err))
		return
	}

	// Update computed fields from API response
	data.ID = types.StringValue(overlay.ID)
	data.OrganizationID = types.StringValue(overlay.OrganizationID)
	data.CreatedBy = types.StringValue(overlay.CreatedBy)
	data.CreatedAt = types.StringValue(overlay.CreatedAt)
	data.UpdatedAt = types.StringValue(overlay.UpdatedAt)

	// Keep the planned data value - API returns same content but with different key ordering
	// data.Data is already set from the plan, no need to update it

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OverlayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OverlayResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	overlay, err := r.client.GetOverlay(data.ID.ValueString())
	if err != nil {
		// If 404, remove from state
		if err.Error() == "API error 404: Not Found" || (len(err.Error()) > 13 && err.Error()[0:13] == "API error 404") {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read overlay, got error: %s", err))
		return
	}

	data.Name = types.StringValue(overlay.Name)
	// Store null instead of empty string for description (to match config when unset)
	if overlay.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(overlay.Description)
	}
	data.OrganizationID = types.StringValue(overlay.OrganizationID)
	data.CreatedBy = types.StringValue(overlay.CreatedBy)
	data.CreatedAt = types.StringValue(overlay.CreatedAt)
	data.UpdatedAt = types.StringValue(overlay.UpdatedAt)

	// Only update data if semantically different (API returns different key ordering)
	if !jsonEqual(data.Data.ValueString(), string(overlay.Data)) {
		data.Data = types.StringValue(string(overlay.Data))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// stringEqualOrBothEmpty returns true if both values are equal, or both are "empty" (null or "")
func stringEqualOrBothEmpty(a, b types.String) bool {
	aEmpty := a.IsNull() || a.ValueString() == ""
	bEmpty := b.IsNull() || b.ValueString() == ""
	if aEmpty && bEmpty {
		return true
	}
	return a.Equal(b)
}

// jsonEqual compares two JSON strings for semantic equality (ignoring key order)
func jsonEqual(a, b string) bool {
	var objA, objB interface{}
	if err := json.Unmarshal([]byte(a), &objA); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &objB); err != nil {
		return false
	}
	return deepEqual(objA, objB)
}

// deepEqual recursively compares two values for equality
func deepEqual(a, b interface{}) bool {
	switch va := a.(type) {
	case map[string]interface{}:
		vb, ok := b.(map[string]interface{})
		if !ok || len(va) != len(vb) {
			return false
		}
		for k, valA := range va {
			valB, exists := vb[k]
			if !exists || !deepEqual(valA, valB) {
				return false
			}
		}
		return true
	case []interface{}:
		vb, ok := b.([]interface{})
		if !ok || len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !deepEqual(va[i], vb[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func (r *OverlayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OverlayResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var rawData json.RawMessage
	if err := json.Unmarshal([]byte(data.Data.ValueString()), &rawData); err != nil {
		resp.Diagnostics.AddError("Invalid JSON in data", err.Error())
		return
	}

	payload := client.OverlayPayload{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Data:        rawData,
	}

	overlay, err := r.client.UpdateOverlay(data.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update overlay, got error: %s", err))
		return
	}

	// Update computed fields from API response
	data.OrganizationID = types.StringValue(overlay.OrganizationID)
	data.CreatedBy = types.StringValue(overlay.CreatedBy)
	data.CreatedAt = types.StringValue(overlay.CreatedAt)
	data.UpdatedAt = types.StringValue(overlay.UpdatedAt)

	// Keep the planned data value - API returns same content but with different key ordering
	// data.Data is already set from the plan, no need to update it

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OverlayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OverlayResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOverlay(data.ID.ValueString())
	if err != nil {
		// If 404, treat as success?
		if len(err.Error()) > 13 && err.Error()[0:13] == "API error 404" {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete overlay, got error: %s", err))
		return
	}
}

func (r *OverlayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID

	// Try to get overlay by ID first
	overlay, err := r.client.GetOverlay(id)
	if err != nil {
		// If failed, try to get by name
		overlay, err = r.client.GetOverlayByName(id)
		if err != nil {
			resp.Diagnostics.AddError(
				"Import Error",
				fmt.Sprintf("Unable to import overlay. Tried as ID and name, got error: %s", err),
			)
			return
		}
	}

	// Set all state attributes from the fetched overlay
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), overlay.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), overlay.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), overlay.Description)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), overlay.OrganizationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("created_by"), overlay.CreatedBy)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("created_at"), overlay.CreatedAt)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("updated_at"), overlay.UpdatedAt)...)

	// Normalize JSON data
	dataBytes, _ := json.Marshal(overlay.Data)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("data"), string(dataBytes))...)
}
