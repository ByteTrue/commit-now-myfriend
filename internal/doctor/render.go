package doctor

import "fmt"

func RenderReport(report Report) string {
	return fmt.Sprintf(
		"cnm doctor\n\nStatus: %s\nErrors: %d\nWarnings: %d\n\nGit: %s\nRepository: %s\nEffective config: %s\nProvider capability: %s\n",
		report.Status,
		report.Summary.Errors,
		report.Summary.Warnings,
		report.Checks.Git.Message,
		report.Checks.Repository.Message,
		report.Checks.EffectiveConfig.Message,
		report.Checks.ProviderCapability.Message,
	)
}
