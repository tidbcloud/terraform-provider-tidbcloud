package provider

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// rootPasswordValidator leaves null and unknown values for the resource to
// handle, since existing clusters do not need a password in configuration.
type rootPasswordValidator struct{}

func (rootPasswordValidator) Description(context.Context) string {
	return "The root password must be between 8 and 64 characters."
}

func (v rootPasswordValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (rootPasswordValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if err := validateRootPassword(req.ConfigValue, false); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid root password", err.Error())
	}
}

func validateRootPassword(password types.String, required bool) error {
	if password.IsUnknown() {
		return errors.New("root_password must be known before applying the configuration")
	}
	if password.IsNull() {
		if required {
			return errors.New("config.root_password is required when creating a tidbcloud_cluster")
		}
		return nil
	}
	if length := utf8.RuneCountInString(password.ValueString()); length < 8 || length > 64 {
		return errors.New("root_password must be between 8 and 64 characters")
	}
	return nil
}
