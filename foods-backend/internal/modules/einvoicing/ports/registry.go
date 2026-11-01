package ports

// Registry resolves jurisdictions by ISO country code and providers by key.
// cmd/api builds it; a country without a jurisdiction cannot issue.
type Registry struct {
	jurisdictions map[string]Jurisdiction
	providers     map[string]Provider
	order         []string
}

func NewRegistry() *Registry {
	return &Registry{jurisdictions: map[string]Jurisdiction{}, providers: map[string]Provider{}}
}
func (r *Registry) WithJurisdiction(j Jurisdiction) *Registry {
	r.jurisdictions[j.Country()] = j
	return r
}
func (r *Registry) WithProvider(p Provider) *Registry {
	if _, ok := r.providers[p.Key()]; !ok {
		r.order = append(r.order, p.Key())
	}
	r.providers[p.Key()] = p
	return r
}
func (r *Registry) Jurisdiction(country string) (Jurisdiction, bool) {
	j, ok := r.jurisdictions[country]
	return j, ok
}
func (r *Registry) Provider(key string) (Provider, bool) {
	p, ok := r.providers[key]
	return p, ok
}
func (r *Registry) ProviderOptions() []Option {
	out := make([]Option, 0, len(r.order))
	for _, key := range r.order {
		out = append(out, Option{Value: key, Label: r.providers[key].Label()})
	}
	return out
}
