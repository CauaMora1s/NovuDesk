package billing

import (
	"context"

	"github.com/novudesk/novudesk/internal/domain/billing"
)

// Checkout is the result of asking a provider to start a payment for a session.
type Checkout struct {
	ProviderRef string // external reference (e.g. Stripe checkout session id)
	URL         string // redirect URL when the provider hosts checkout; empty for manual
}

// Provider abstracts the payment backend. A real implementation (e.g. Stripe)
// would create a hosted checkout and confirm via webhook; the manual provider
// requires an explicit in-app confirmation.
type Provider interface {
	Name() string
	CreateCheckout(ctx context.Context, session *billing.PaymentSession) (Checkout, error)
}

// ManualProvider is the default no-gateway provider. It records the intent and
// relies on an explicit confirmation step to apply the plan change.
type ManualProvider struct{}

func NewManualProvider() *ManualProvider { return &ManualProvider{} }

func (p *ManualProvider) Name() string { return "manual" }

func (p *ManualProvider) CreateCheckout(_ context.Context, _ *billing.PaymentSession) (Checkout, error) {
	return Checkout{}, nil
}
