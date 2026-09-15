// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"slices"
	"testing"

	"github.com/hashicorp/terraform-provider-google/google/fwprovider"
	"github.com/hashicorp/terraform-provider-google/google/provider"
)

// resourcesReadingWriteOnlyVersion lists the Terraform resources whose
// generated Read() calls d.Set on a write-only version attribute. For these,
// the attribute has to stay in the runtime schema or every operation on the
// resource fails with "Invalid address to set".
//
// Keep this in sync with:
//
//	grep -rn 'd\.Set("[a-z_]*_wo_version"' google/services/ (terraform-provider-google)
var resourcesReadingWriteOnlyVersion = map[string]string{
	"google_compute_region_ssl_certificate": "private_key_wo_version",
	"google_compute_ssl_certificate":        "private_key_wo_version",
	"google_compute_vpn_tunnel":             "shared_secret_wo_version",
}

// TestWriteOnlyFieldsPresentAtRuntime asserts that the write-only attributes
// whose values the Terraform provider writes back during Read() are still in
// the runtime schema. Removing them makes create, observe and delete fail and
// strands the managed resource behind its finalizer.
func TestWriteOnlyFieldsPresentAtRuntime(t *testing.T) {
	sdkProvider := provider.Provider()
	pc, err := GetProvider(t.Context(), sdkProvider, fwprovider.New(sdkProvider), false)
	if err != nil {
		t.Fatalf("GetProvider: %s", err)
	}

	for name, field := range resourcesReadingWriteOnlyVersion {
		r, ok := pc.Resources[name]
		if !ok {
			t.Fatalf("resource %q is not configured", name)
		}
		if _, ok := r.TerraformResource.Schema[field]; !ok {
			t.Errorf("resource %q: runtime schema is missing %q; the Terraform provider's Read() "+
				"sets this key, so d.Set will fail with \"Invalid address to set\"", name, field)
		}
	}
}

// TestWriteOnlyFieldsAbsentFromGeneratedAPI asserts that no write-only
// attribute reaches the schema the CRDs are generated from. These values are
// accepted through SecretRef parameters, so surfacing the plaintext attribute
// would let a secret be stored in the API server.
func TestWriteOnlyFieldsAbsentFromGeneratedAPI(t *testing.T) {
	sdkProvider := provider.Provider()
	pc, err := GetProvider(t.Context(), sdkProvider, fwprovider.New(sdkProvider), true)
	if err != nil {
		t.Fatalf("GetProvider: %s", err)
	}

	for name, fields := range writeOnlyFields {
		r, ok := pc.Resources[name]
		if !ok {
			t.Fatalf("resource %q is not configured", name)
		}
		for _, f := range fields {
			if _, ok := r.TerraformResource.Schema[f]; ok {
				t.Errorf("resource %q: %q is present in the generation schema and would be "+
					"rendered into the CRD in plaintext", name, f)
			}
		}
	}
}

// TestWriteOnlyFieldsAreEnumerated asserts that writeOnlyFields covers every
// top-level write-only attribute the Terraform provider declares on a
// configured resource, so that a bump of terraform-provider-google cannot
// introduce one that silently reaches the CRDs in plaintext.
//
// The check reads the runtime schema deliberately. The schema used for code
// generation is rebuilt from config/schema.json, whose JSON representation
// does not carry the write-only marker, so the flag is only observable on the
// Go schema the Terraform provider itself defines.
//
// The walk is deliberately limited to top-level attributes, which is the set
// writeOnlyFields addresses. Nested write-only attributes are not covered.
func TestWriteOnlyFieldsAreEnumerated(t *testing.T) {
	sdkProvider := provider.Provider()
	pc, err := GetProvider(t.Context(), sdkProvider, fwprovider.New(sdkProvider), false)
	if err != nil {
		t.Fatalf("GetProvider: %s", err)
	}

	for name, r := range pc.Resources {
		if r.TerraformResource == nil {
			continue
		}
		for f, s := range r.TerraformResource.Schema {
			if !s.WriteOnly {
				continue
			}
			if !slices.Contains(writeOnlyFields[name], f) {
				t.Errorf("resource %q declares write-only attribute %q, which is not listed in "+
					"writeOnlyFields; it would be rendered into the CRD in plaintext", name, f)
			}
		}
	}
}
