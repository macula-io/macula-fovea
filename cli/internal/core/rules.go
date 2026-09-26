package core

// Rule is one lint rule. Its code is what findings, conformance cases and
// docs name. A Rule can only be made by newRule inside the table below, so
// every rule the lint can raise is in allRules, and the conformance test
// that iterates allRules sees every one of them.
type Rule struct{ code string }

// Code is the stable rule code, e.g. "grid_missing_column".
func (r Rule) Code() string { return r.code }

var allRules []Rule

func newRule(code string) Rule {
	r := Rule{code}
	allRules = append(allRules, r)
	return r
}

// The rule table: the only place rule codes are spelled.
var (
	ruleGridMissingCell               = newRule("grid_missing_cell")
	ruleCellIDUndeclared              = newRule("cell_id_undeclared")
	ruleCellStatusUnknown             = newRule("cell_status_unknown")
	ruleCellOwnerUnassigned           = newRule("cell_owner_unassigned")
	ruleCellUnassessed                = newRule("cell_unassessed")
	ruleNAWithoutReason               = newRule("na_without_reason")
	ruleDefinitionEmpty               = newRule("definition_empty")
	ruleManifestationsEmpty           = newRule("manifestations_empty")
	ruleMeasuresEmpty                 = newRule("measures_empty")
	ruleDetectionEmpty                = newRule("detection_empty")
	ruleReviewByNotISODate            = newRule("review_by_not_iso_date")
	ruleRoadmapWithoutReviewBy        = newRule("roadmap_without_review_by")
	ruleRoadmapMeasureWithoutReviewBy = newRule("roadmap_measure_without_review_by")
	ruleMeasureStatusUnknown          = newRule("measure_status_unknown")
	ruleByDesignWithoutSource         = newRule("by_design_without_source")
	ruleAssessedAllOrg                = newRule("assessed_all_org")
	ruleDefinitionCopyPaste           = newRule("definition_copy_paste")
	ruleHeaderVersionUnknown          = newRule("header_version_unknown")
	ruleHeaderOwnerUnassigned         = newRule("header_owner_unassigned")
	ruleGridFamilyUnknown             = newRule("grid_family_unknown")
	ruleGridColumnDuplicate           = newRule("grid_column_duplicate")
	ruleGridColumnWrongFamily         = newRule("grid_column_wrong_family")
	ruleGridColumnUnknown             = newRule("grid_column_unknown")
	ruleGridMissingColumn             = newRule("grid_missing_column")
	ruleGridAttributeDuplicate        = newRule("grid_attribute_duplicate")
	ruleGridCoreAttributeInvalid      = newRule("grid_core_attribute_invalid")
	ruleGridAttributeUnknown          = newRule("grid_attribute_unknown")
	ruleGridCoreAttributeMissing      = newRule("grid_core_attribute_missing")
	ruleGridExtensionContradiction    = newRule("grid_extension_contradiction")
	ruleGridExtensionUnjustified      = newRule("grid_extension_unjustified")
	ruleCellsDirUnreadable            = newRule("cells_dir_unreadable")
	ruleCellParse                     = newRule("cell_parse")
	ruleCellIDFilenameMismatch        = newRule("cell_id_filename_mismatch")
)
