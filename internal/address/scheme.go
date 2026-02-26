package address

// Network identifies which overlay network an address belongs to.
type Network int

const (
	NetworkI2P Network = iota
	NetworkTorV3
)

func (n Network) String() string {
	switch n {
	case NetworkI2P:
		return "i2p"
	case NetworkTorV3:
		return "torv3"
	default:
		return "unknown"
	}
}

// ParseNetwork converts a persisted string back to a Network constant.
func ParseNetwork(s string) Network {
	switch s {
	case "torv3":
		return NetworkTorV3
	default:
		return NetworkI2P
	}
}

// Candidate represents a generated keypair and its associated address.
type Candidate interface {
	// Address returns the base32 address without the network suffix.
	Address() string
	// FullAddress returns the complete address with suffix (.b32.i2p or .onion).
	FullAddress() string
	// SaveKeys writes the private keys to disk in the appropriate format.
	SaveKeys(path string) error
}

// Scheme defines the address generation strategy for a specific network.
type Scheme interface {
	Network() Network
	Suffix() string
	ValidatePrefix(prefix string) error
	EstimateAttempts(prefixLen int) float64
	MaxPrefixLen() int
	NewCandidate() (Candidate, error)
	SupportsGPU() bool
}
