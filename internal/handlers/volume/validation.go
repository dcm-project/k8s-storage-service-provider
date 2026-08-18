package volume

import (
	"fmt"
	"regexp"

	v1alpha1 "github.com/dcm-project/k8s-storage-service-provider/api/v1alpha1"
	"github.com/dcm-project/k8s-storage-service-provider/internal/dcm"
)

// reservedVolumeIDs cannot be used because they collide with fixed API paths
// under /api/v1alpha1/volumes/.
var reservedVolumeIDs = map[string]bool{
	"health": true,
}

var aep122IDPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func validateVolumeID(id string) error {
	if reservedVolumeIDs[id] {
		return fmt.Errorf("volume ID %q is reserved and cannot be used", id)
	}
	if !aep122IDPattern.MatchString(id) {
		return fmt.Errorf("volume ID %q must match AEP-122 pattern", id)
	}
	return nil
}

func validateCreateSpec(spec v1alpha1.StorageSpec) error {
	if spec.ServiceType != v1alpha1.Storage {
		return fmt.Errorf("service_type must be %q", v1alpha1.Storage)
	}
	if spec.Capacity == "" {
		return fmt.Errorf("capacity is required")
	}
	if spec.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	return validateVolumeID(spec.Metadata.Name)
}

func validateUserLabels(labels *map[string]string) error {
	if labels == nil {
		return nil
	}
	for k := range *labels {
		if dcm.ReservedLabelKeys[k] {
			return fmt.Errorf("label %q is reserved by DCM and cannot be set by the user", k)
		}
	}
	return nil
}
