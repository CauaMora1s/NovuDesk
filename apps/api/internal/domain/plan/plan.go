package plan

// Tier identifies a subscription tier. Tiers are static, code-defined config so
// the catalog can evolve without a database migration.
type Tier string

const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierBusiness   Tier = "business"
	TierEnterprise Tier = "enterprise"
)

// Unlimited is the sentinel value for a limit with no cap.
const Unlimited int64 = -1

// IsUnlimited reports whether a limit value means "no cap".
func IsUnlimited(v int64) bool { return v < 0 }

// Limits holds the per-tier usage caps. A value of Unlimited (-1) means no cap.
type Limits struct {
	Seats           int64 `json:"seats"`
	TicketsPerMonth int64 `json:"tickets_per_month"`
	StorageBytes    int64 `json:"storage_bytes"`
	Teams           int64 `json:"teams"`
	Categories      int64 `json:"categories"`
	APIKeys         int64 `json:"api_keys"`
}

// Plan describes a subscription tier and its limits.
type Plan struct {
	Tier       Tier   `json:"tier"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"` // monthly price; 0 for free / custom (enterprise)
	Currency   string `json:"currency"`
	Limits     Limits `json:"limits"`
}

const (
	mb = int64(1024 * 1024)
	gb = 1024 * mb
)

// catalog is the ordered list of available plans, cheapest first.
var catalog = []Plan{
	{
		Tier: TierFree, Name: "Free", PriceCents: 0, Currency: "BRL",
		Limits: Limits{Seats: 3, TicketsPerMonth: 100, StorageBytes: 500 * mb, Teams: 1, Categories: 5, APIKeys: 1},
	},
	{
		Tier: TierPro, Name: "Pro", PriceCents: 9900, Currency: "BRL",
		Limits: Limits{Seats: 15, TicketsPerMonth: 2000, StorageBytes: 10 * gb, Teams: 10, Categories: 50, APIKeys: 10},
	},
	{
		Tier: TierBusiness, Name: "Business", PriceCents: 49900, Currency: "BRL",
		Limits: Limits{Seats: 50, TicketsPerMonth: 20000, StorageBytes: 100 * gb, Teams: Unlimited, Categories: Unlimited, APIKeys: 50},
	},
	{
		Tier: TierEnterprise, Name: "Enterprise", PriceCents: 0, Currency: "BRL",
		Limits: Limits{Seats: Unlimited, TicketsPerMonth: Unlimited, StorageBytes: Unlimited, Teams: Unlimited, Categories: Unlimited, APIKeys: Unlimited},
	},
}

// Catalog returns all available plans, cheapest first.
func Catalog() []Plan {
	out := make([]Plan, len(catalog))
	copy(out, catalog)
	return out
}

// ByTier returns the plan for a tier, or false if the tier is unknown.
func ByTier(tier Tier) (Plan, bool) {
	for _, p := range catalog {
		if p.Tier == tier {
			return p, true
		}
	}
	return Plan{}, false
}

// IsValidTier reports whether the given string is a known tier.
func IsValidTier(tier string) bool {
	_, ok := ByTier(Tier(tier))
	return ok
}

// Proration returns the prorated charge (in cents) for switching from one plan
// to another partway through a billing cycle. Only upgrades are charged; a
// downgrade returns 0 (the change applies but no immediate charge is taken).
func Proration(from, to Plan, daysRemaining, cycleDays int) int64 {
	if cycleDays <= 0 {
		cycleDays = 30
	}
	if daysRemaining < 0 {
		daysRemaining = 0
	}
	if daysRemaining > cycleDays {
		daysRemaining = cycleDays
	}
	diff := to.PriceCents - from.PriceCents
	if diff <= 0 {
		return 0
	}
	return diff * int64(daysRemaining) / int64(cycleDays)
}
