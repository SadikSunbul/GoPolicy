package policy

// PolicyState represents policy states.
type PolicyState int

const (
	PolicyStateNotConfigured PolicyState = 0
	PolicyStateDisabled      PolicyState = 1
	PolicyStateEnabled       PolicyState = 2
	PolicyStateUnknown       PolicyState = 3
)

// String returns the readable value of a PolicyState.
func (ps PolicyState) String() string {
	switch ps {
	case PolicyStateNotConfigured:
		return "Not Configured"
	case PolicyStateDisabled:
		return "Disabled"
	case PolicyStateEnabled:
		return "Enabled"
	default:
		return "Unknown"
	}
}
