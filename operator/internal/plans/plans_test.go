package plans

import (
	"testing"
)

func TestGetPlanQuotas(t *testing.T) {
	tests := []struct {
		name     string
		plan     string
		expected PlanQuotas
	}{
		{
			name: "free plan",
			plan: "free",
			expected: PlanQuotas{
				Keys:               5,
				SignaturesPerMonth: 10000,
				Namespaces:         1,
				TeamMembers:        1,
				APIKeys:            2,
			},
		},
		{
			name: "starter plan",
			plan: "starter",
			expected: PlanQuotas{
				Keys:               10,
				SignaturesPerMonth: 100000,
				Namespaces:         2,
				TeamMembers:        3,
				APIKeys:            5,
			},
		},
		{
			name: "pro plan",
			plan: "pro",
			expected: PlanQuotas{
				Keys:               25,
				SignaturesPerMonth: 500000,
				Namespaces:         5,
				TeamMembers:        10,
				APIKeys:            20,
			},
		},
		{
			name: "enterprise plan",
			plan: "enterprise",
			expected: PlanQuotas{
				Keys:               -1,
				SignaturesPerMonth: -1,
				Namespaces:         -1,
				TeamMembers:        -1,
				APIKeys:            -1,
			},
		},
		{
			name: "unknown plan defaults to free",
			plan: "unknown",
			expected: PlanQuotas{
				Keys:               5,
				SignaturesPerMonth: 10000,
				Namespaces:         1,
				TeamMembers:        1,
				APIKeys:            2,
			},
		},
		{
			name: "empty plan defaults to free",
			plan: "",
			expected: PlanQuotas{
				Keys:               5,
				SignaturesPerMonth: 10000,
				Namespaces:         1,
				TeamMembers:        1,
				APIKeys:            2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPlanQuotas(tt.plan)
			if got != tt.expected {
				t.Errorf("GetPlanQuotas(%q) = %+v, want %+v", tt.plan, got, tt.expected)
			}
		})
	}
}

func TestIsUnlimited(t *testing.T) {
	tests := []struct {
		name     string
		value    int32
		expected bool
	}{
		{"unlimited", -1, true},
		{"zero", 0, false},
		{"positive", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnlimited(tt.value); got != tt.expected {
				t.Errorf("IsUnlimited(%d) = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestIsUnlimitedInt64(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected bool
	}{
		{"unlimited", -1, true},
		{"zero", 0, false},
		{"positive", 10000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnlimitedInt64(tt.value); got != tt.expected {
				t.Errorf("IsUnlimitedInt64(%d) = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestCheckQuota(t *testing.T) {
	tests := []struct {
		name     string
		usage    int32
		quota    int32
		expected bool
	}{
		{"unlimited quota", 1000, -1, true},
		{"within quota", 3, 5, true},
		{"at quota", 5, 5, false},
		{"over quota", 6, 5, false},
		{"zero usage", 0, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckQuota(tt.usage, tt.quota); got != tt.expected {
				t.Errorf("CheckQuota(%d, %d) = %v, want %v", tt.usage, tt.quota, got, tt.expected)
			}
		})
	}
}

func TestCheckQuotaInt64(t *testing.T) {
	tests := []struct {
		name     string
		usage    int64
		quota    int64
		expected bool
	}{
		{"unlimited quota", 1000000, -1, true},
		{"within quota", 9999, 10000, true},
		{"at quota", 10000, 10000, false},
		{"over quota", 10001, 10000, false},
		{"zero usage", 0, 10000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckQuotaInt64(tt.usage, tt.quota); got != tt.expected {
				t.Errorf("CheckQuotaInt64(%d, %d) = %v, want %v", tt.usage, tt.quota, got, tt.expected)
			}
		})
	}
}

func TestValidPlans(t *testing.T) {
	plans := ValidPlans()
	expected := []string{"free", "starter", "pro", "enterprise"}

	if len(plans) != len(expected) {
		t.Errorf("ValidPlans() returned %d plans, want %d", len(plans), len(expected))
	}

	for _, e := range expected {
		found := false
		for _, p := range plans {
			if p == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidPlans() missing plan %q", e)
		}
	}
}

func TestIsValidPlan(t *testing.T) {
	tests := []struct {
		name     string
		plan     string
		expected bool
	}{
		{"free", "free", true},
		{"starter", "starter", true},
		{"pro", "pro", true},
		{"enterprise", "enterprise", true},
		{"invalid", "invalid", false},
		{"empty", "", false},
		{"random", "random-plan", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPlan(tt.plan); got != tt.expected {
				t.Errorf("IsValidPlan(%q) = %v, want %v", tt.plan, got, tt.expected)
			}
		})
	}
}

func TestEnterpriseQuotasAreUnlimited(t *testing.T) {
	quotas := GetPlanQuotas("enterprise")

	if !IsUnlimited(quotas.Keys) {
		t.Error("Enterprise Keys should be unlimited")
	}
	if !IsUnlimitedInt64(quotas.SignaturesPerMonth) {
		t.Error("Enterprise SignaturesPerMonth should be unlimited")
	}
	if !IsUnlimited(quotas.Namespaces) {
		t.Error("Enterprise Namespaces should be unlimited")
	}
	if !IsUnlimited(quotas.TeamMembers) {
		t.Error("Enterprise TeamMembers should be unlimited")
	}
	if !IsUnlimited(quotas.APIKeys) {
		t.Error("Enterprise APIKeys should be unlimited")
	}
}
