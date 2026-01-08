package awsassetinventory

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// ReportGenerator generates markdown reports from inventory data.
type ReportGenerator struct {
	inventory *Inventory
}

// NewReportGenerator creates a new ReportGenerator for the given inventory.
func NewReportGenerator(inv *Inventory) *ReportGenerator {
	return &ReportGenerator{inventory: inv}
}

// Generate writes a complete markdown report to the provided writer.
func (rg *ReportGenerator) Generate(w io.Writer) error {
	if err := rg.writeHeader(w); err != nil {
		return err
	}
	if err := rg.writeSummary(w); err != nil {
		return err
	}
	if err := rg.writeByRegion(w); err != nil {
		return err
	}
	if err := rg.writeResourceDetails(w); err != nil {
		return err
	}
	return nil
}

func (rg *ReportGenerator) writeHeader(w io.Writer) error {
	_, err := fmt.Fprintf(w, "# AWS Asset Inventory Report\n\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "**Collected:** %s\n", rg.inventory.CollectedAt.Format("2006-01-02 15:04:05 UTC"))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "**Profile:** %s\n", rg.inventory.Profile)
	if err != nil {
		return err
	}

	regionStrings := make([]string, len(rg.inventory.Regions))
	for i, r := range rg.inventory.Regions {
		regionStrings[i] = r.String()
	}
	_, err = fmt.Fprintf(w, "**Regions:** %s\n", strings.Join(regionStrings, ", "))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "**Total Resources:** %d\n\n", rg.inventory.ResourceCount())
	return err
}

func (rg *ReportGenerator) writeSummary(w io.Writer) error {
	_, err := fmt.Fprintf(w, "## Summary\n\n")
	if err != nil {
		return err
	}

	counts := rg.inventory.ResourceCountByType()
	if len(counts) == 0 {
		_, err = fmt.Fprintf(w, "No resources found.\n\n")
		return err
	}

	_, err = fmt.Fprintf(w, "| Resource Type | Count |\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "|---------------|-------|\n")
	if err != nil {
		return err
	}

	sortedTypes := sortedResourceTypes(counts)
	for _, rt := range sortedTypes {
		_, err = fmt.Fprintf(w, "| %s | %d |\n", rt, counts[rt])
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, "\n")
	return err
}

func (rg *ReportGenerator) writeByRegion(w io.Writer) error {
	_, err := fmt.Fprintf(w, "## By Region\n\n")
	if err != nil {
		return err
	}

	countsByRegion := rg.inventory.ResourceCountByTypeAndRegion()
	if len(countsByRegion) == 0 {
		return nil
	}

	sortedRegions := sortedRegions(countsByRegion)
	for _, region := range sortedRegions {
		typeCounts := countsByRegion[region]

		_, err = fmt.Fprintf(w, "### %s\n\n", region)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, "| Resource Type | Count |\n")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "|---------------|-------|\n")
		if err != nil {
			return err
		}

		sortedTypes := sortedResourceTypes(typeCounts)
		for _, rt := range sortedTypes {
			_, err = fmt.Fprintf(w, "| %s | %d |\n", rt, typeCounts[rt])
			if err != nil {
				return err
			}
		}

		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func (rg *ReportGenerator) writeResourceDetails(w io.Writer) error {
	_, err := fmt.Fprintf(w, "## Resource Details\n\n")
	if err != nil {
		return err
	}

	grouped := rg.inventory.ResourcesByType()
	if len(grouped) == 0 {
		_, err = fmt.Fprintf(w, "No resources to display.\n\n")
		return err
	}

	counts := rg.inventory.ResourceCountByType()
	sortedTypes := sortedResourceTypes(counts)

	for _, rt := range sortedTypes {
		resources := grouped[rt]

		_, err = fmt.Fprintf(w, "### %s (%d)\n\n", rt, len(resources))
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, "| Name | ID | Region | ARN |\n")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "|------|----|----|-----|\n")
		if err != nil {
			return err
		}

		for _, r := range resources {
			name := r.ResourceName
			if name == "" {
				name = "-"
			}
			arn := r.ARN
			if arn == "" {
				arn = "-"
			}
			_, err = fmt.Fprintf(w, "| %s | %s | %s | %s |\n",
				escapeMarkdown(name),
				escapeMarkdown(r.ResourceID),
				r.Region,
				escapeMarkdown(truncateARN(arn)))
			if err != nil {
				return err
			}
		}

		_, err = fmt.Fprintf(w, "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func sortedResourceTypes(counts map[ResourceType]int) []ResourceType {
	types := make([]ResourceType, 0, len(counts))
	for rt := range counts {
		types = append(types, rt)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})
	return types
}

func sortedRegions(countsByRegion map[Region]map[ResourceType]int) []Region {
	regions := make([]Region, 0, len(countsByRegion))
	for r := range countsByRegion {
		regions = append(regions, r)
	}
	sort.Slice(regions, func(i, j int) bool {
		return regions[i] < regions[j]
	})
	return regions
}

func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func truncateARN(arn string) string {
	const maxLen = 60
	if len(arn) <= maxLen {
		return arn
	}
	return arn[:maxLen-3] + "..."
}
