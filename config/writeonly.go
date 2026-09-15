// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
)

// writeOnlyFields lists, per Terraform resource, the write-only attributes
// that must not appear in the generated CRD API. Crossplane accepts these
// values through the corresponding SecretRef parameters instead, so exposing
// the plaintext attributes would let a secret be stored in the API server.
//
// Every entry is removed from the schema used for code generation only. See
// deleteWriteOnlyFields for why they must survive at runtime.
var writeOnlyFields = map[string][]string{
	"google_compute_region_ssl_certificate": {"private_key_wo", "private_key_wo_version"},
	"google_compute_ssl_certificate":        {"private_key_wo", "private_key_wo_version"},
	"google_compute_vpn_tunnel":             {"shared_secret_wo", "shared_secret_wo_version"},
	"google_secret_manager_secret_version":  {"secret_data_wo", "secret_data_wo_version"},
	"google_sql_database_instance":          {"root_password_wo", "root_password_wo_version"},
	"google_sql_user":                       {"password_wo", "password_wo_version"},
}

// deleteWriteOnlyFields removes the write-only attributes listed in
// writeOnlyFields from the Terraform schemas of the provider's resources.
//
// The attributes must only be removed when generating the API types. The
// Read() implementations that terraform-provider-google generates for some of
// these resources call d.Set("<field>_wo_version", ...) unconditionally, and
// helper/schema's d.Set reports "Invalid address to set" when the key is
// absent from the runtime schema. Read() runs during create, observe and
// delete alike, so removing the attribute at runtime fails every operation on
// the resource and leaves the managed resource stuck behind its finalizer.
//
// Keeping the two schemas deliberately asymmetric avoids that: the generated
// CRD carries no write-only field, while the runtime schema retains one for
// the Terraform provider to write into. The retained attribute is not mapped
// into the CRD, so no plaintext value can be accepted or persisted through
// the Crossplane API.
//
// This must be called after the resource configurators have run, so that a
// configurator cannot reintroduce one of these attributes.
func deleteWriteOnlyFields(pc *ujconfig.Provider, generationProvider bool) {
	if !generationProvider {
		return
	}
	for name, fields := range writeOnlyFields {
		r, ok := pc.Resources[name]
		if !ok || r.TerraformResource == nil {
			continue
		}
		for _, f := range fields {
			delete(r.TerraformResource.Schema, f)
		}
	}
}
