package handel

// ReportHandel holds a handel struct but modifies it so it is able to issue
// some stats.
type ReportHandel struct {
	*Handel
}

// Reporter is simply a copy of monitor.Counter interface to avoid importint
// monitor in handel.
type Reporter interface {
	Values() map[string]float64
}

// NewReportHandel returns a Handel that can report some statistis about its
// internals
func NewReportHandel(h *Handel) *ReportHandel {
	h.store = newReportStore(h.store)
	return &ReportHandel{h}
}

// Values returns the values of ALL internal components of Handel merged together.
func (r *ReportHandel) Values() map[string]float64 {
	net := r.Network()
	netValues := net.Values()
	store := r.Handel.store.(*ReportStore)
	storeValues := store.Values()
	merged := make(map[string]float64)
	for k, v := range netValues {
		merged["net_"+k] = float64(v)
	}
	for k, v := range storeValues {
		merged["store_"+k] = float64(v)
	}
	return merged
}

// Network returns the Network reporter interface
func (r *ReportHandel) Network() Reporter {
	return r.Handel.net.(Reporter)
}

// Store returns the Store reporter interface
func (r *ReportHandel) Store() Reporter {
	return r.Handel.store.(*ReportStore)
}

// Processing returns the Store reporter interface
func (r *ReportHandel) Processing() Reporter {
	return r.Handel.proc.(Reporter)
}

// ReportStore is a Store that can report some statistics about the storage
type ReportStore struct {
	signatureStore
	sucessReplaced int64
	replacedTrial  int64
}

// newReportStore returns a signatureStore with som eadditional reporting
// capabilities
func newReportStore(s signatureStore) signatureStore {
	return &ReportStore{
		signatureStore: s,
	}
}

// Store implements the signatureStore interface
func (r *ReportStore) Store(sp *incomingSig) *MultiSignature {
	ms := r.signatureStore.Store(sp)
	if ms != nil {
		r.sucessReplaced++
	} else {
		r.replacedTrial++
	}
	return ms
}

// Values implements the simul/monitor/counterIO interface
func (r *ReportStore) Values() map[string]float64 {
	return map[string]float64{
		"successReplace": float64(r.sucessReplaced),
		"replaceTrial":   float64(r.replacedTrial),
	}
}
