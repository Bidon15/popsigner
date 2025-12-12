package plans

// PlanQuotas defines the resource quotas for a plan
type PlanQuotas struct {
	Keys               int32
	SignaturesPerMonth int64
	Namespaces         int32
	TeamMembers        int32
	APIKeys            int32
}

// Plans is a map of plan names to their quotas
// -1 values indicate unlimited resources
var Plans = map[string]PlanQuotas{
	"free": {
		Keys:               5,
		SignaturesPerMonth: 10000,
		Namespaces:         1,
		TeamMembers:        1,
		APIKeys:            2,
	},
	"starter": {
		Keys:               10,
		SignaturesPerMonth: 100000,
		Namespaces:         2,
		TeamMembers:        3,
		APIKeys:            5,
	},
	"pro": {
		Keys:               25,
		SignaturesPerMonth: 500000,
		Namespaces:         5,
		TeamMembers:        10,
		APIKeys:            20,
	},
	"enterprise": {
		Keys:               -1, // unlimited
		SignaturesPerMonth: -1,
		Namespaces:         -1,
		TeamMembers:        -1,
		APIKeys:            -1,
	},
}

// GetPlanQuotas returns the quotas for a given plan
// Returns "free" plan quotas if the plan is not found
func GetPlanQuotas(plan string) PlanQuotas {
	if q, ok := Plans[plan]; ok {
		return q
	}
	return Plans["free"]
}

// IsUnlimited checks if a quota value is unlimited (-1)
func IsUnlimited(value int32) bool {
	return value == -1
}

// IsUnlimitedInt64 checks if a quota value is unlimited (-1)
func IsUnlimitedInt64(value int64) bool {
	return value == -1
}

// CheckQuota checks if usage is within quota
// Returns true if within quota, false if exceeded
func CheckQuota(usage, quota int32) bool {
	if IsUnlimited(quota) {
		return true
	}
	return usage < quota
}

// CheckQuotaInt64 checks if usage is within quota for int64 values
func CheckQuotaInt64(usage, quota int64) bool {
	if IsUnlimitedInt64(quota) {
		return true
	}
	return usage < quota
}

// ValidPlans returns a list of valid plan names
func ValidPlans() []string {
	return []string{"free", "starter", "pro", "enterprise"}
}

// IsValidPlan checks if a plan name is valid
func IsValidPlan(plan string) bool {
	_, ok := Plans[plan]
	return ok
}
