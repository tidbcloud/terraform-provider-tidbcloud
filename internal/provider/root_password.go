package provider

import (
	"errors"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
