package exam

// Objective is one line of the Terraform Associate 004 content list.
type Objective struct{ ID, Title string }

// Objectives is the content list, as published at
// developer.hashicorp.com/terraform/tutorials/certification-004/associate-review-004
// (read 2026-11-15).
var Objectives = []Objective{
	{"1a", "Explain what IaC is"},
	{"1b", "Describe the advantages of IaC patterns"},
	{"1c", "Multi-cloud, hybrid cloud and service-agnostic workflows"},
	{"2a", "Install and version providers"},
	{"2b", "Describe how Terraform uses providers"},
	{"2c", "Write configuration using multiple providers"},
	{"2d", "How Terraform uses and manages state"},
	{"3a", "Describe the Terraform workflow"},
	{"3b", "Initialize a working directory"},
	{"3c", "Validate a configuration"},
	{"3d", "Generate and review an execution plan"},
	{"3e", "Apply changes to infrastructure"},
	{"3f", "Destroy infrastructure"},
	{"3g", "Formatting and style"},
	{"4a", "Use and differentiate resource and data blocks"},
	{"4b", "Resource attributes and cross-resource references"},
	{"4c", "Use variables and outputs"},
	{"4d", "Understand and use complex types"},
	{"4e", "Dynamic configuration: expressions and functions"},
	{"4f", "Define resource dependencies"},
	{"4g", "Validate with custom conditions"},
	{"4h", "Sensitive data and secrets management"},
	{"5a", "How Terraform sources modules"},
	{"5b", "Variable scope within modules"},
	{"5c", "Use modules in configuration"},
	{"5d", "Manage module versions"},
	{"6a", "Describe the local backend"},
	{"6b", "Describe state locking"},
	{"6c", "Configure remote state with the backend block"},
	{"6d", "Manage resource drift and state"},
	{"7a", "Import existing infrastructure"},
	{"7b", "Use the CLI to inspect state"},
	{"7c", "Verbose logging"},
	{"8a", "Use HCP Terraform to create infrastructure"},
	{"8b", "HCP Terraform collaboration and governance"},
	{"8c", "Organize HCP Terraform workspaces and projects"},
	{"8d", "Configure HCP Terraform integration"},
}
