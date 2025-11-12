package customvalidators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// NamespaceNotReservedPrefixValidator ensures the namespace part of <namespace>/<name>:<port>
// does not start with cluster-, kube-, or default.
type NamespaceNotReservedPrefixValidator struct{}

func NamespaceNotReservedPrefix() NamespaceNotReservedPrefixValidator {
	return NamespaceNotReservedPrefixValidator{}
}

func (v NamespaceNotReservedPrefixValidator) Description(_ context.Context) string {
	return "The namespace must not start with 'cluster-', 'kube-', or 'default'."
}

func (v NamespaceNotReservedPrefixValidator) MarkdownDescription(_ context.Context) string {
	return "The namespace must not start with cluster-, kube-, or default."
}

func (v NamespaceNotReservedPrefixValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return
	}

	ns := parts[0]
	badPrefixes := []string{"cluster-", "kube-", "default"}
	for _, prefix := range badPrefixes {
		if strings.HasPrefix(ns, prefix) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Not valid namespace prefix",
				fmt.Sprintf("Namespace '%s' must not start with '%s'", ns, prefix),
			)
			return
		}
	}
}
