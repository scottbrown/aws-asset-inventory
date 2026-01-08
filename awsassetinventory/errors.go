package awsassetinventory

import (
	"fmt"
	"strings"
)

// RegionError represents an error that occurred in a specific region.
type RegionError struct {
	Region Region
	Err    error
}

func (re RegionError) Error() string {
	return fmt.Sprintf("[%s] %v", re.Region, re.Err)
}

func (re RegionError) Unwrap() error {
	return re.Err
}

// CollectErrors aggregates multiple region errors.
type CollectErrors struct {
	Errors []RegionError
}

func (ce CollectErrors) Error() string {
	if len(ce.Errors) == 1 {
		return ce.Errors[0].Error()
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d regions failed: ", len(ce.Errors)))
	for i, e := range ce.Errors {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(e.Error())
	}
	return sb.String()
}

// Regions returns the list of failed regions.
func (ce CollectErrors) Regions() []Region {
	regions := make([]Region, len(ce.Errors))
	for i, e := range ce.Errors {
		regions[i] = e.Region
	}
	return regions
}
