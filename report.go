package handel

// ReportHandel holds a handel struct but modifies it so it is able to issue
// some stats.
type ReportHandel struct {
	*Handel
}

// Reporter is a generic interface that can report different data about its
// internal state
type Reporter interface {
	Values() map[string]float64
}

// NewReportHandel returns a Handel that can report some statistis about its
// internals
func NewReportHandel(h *Handel) *ReportHandel {
	h.net = NewReportNetwork(h.net)
	h.store = newReportStore(h.store)
	return &ReportHandel{h}
}

// Values returns the values of ALL internal components of Handel merged together.
func (r *ReportHandel) Values() map[string]float64 {
	net := r.Handel.net.(*ReportNetwork)
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
	return r.Handel.net.(*ReportNetwork)
}

// Store returns the Store reporter interface
func (r *ReportHandel) Store() Reporter {
	return r.Handel.store.(*ReportStore)
}

// Processing returns the Store reporter interface
func (r *ReportHandel) Processing() Reporter {
	return r.Handel.proc.(Reporter)
}

// ReportNetwork is a struct that implements the Network interface by augmenting
// the Network's method with accountability. How many packets received and send
// can be logged.
type ReportNetwork struct {
	Network
	sentPackets uint32
	rcvdPackets uint32
	lis         []Listener
}

// NewReportNetwork returns a Network with reporting capabilities.
func NewReportNetwork(n Network) Network {
	r := &ReportNetwork{
		Network: n,
	}
	n.RegisterListener(r)
	return r
}

// Send implements the Network interface
func (r *ReportNetwork) Send(ids []Identity, p *Packet) {
	r.sentPackets++
	r.Network.Send(ids, p)
}

// RegisterListener implements the Network interface
func (r *ReportNetwork) RegisterListener(l Listener) {
	r.lis = append(r.lis, l)
}

// NewPacket implements the Listener interface
func (r *ReportNetwork) NewPacket(p *Packet) {
	r.rcvdPackets++
	for _, l := range r.lis {
		l.NewPacket(p)
	}
}

// Sent returns the number of sent packets
func (r *ReportNetwork) Sent() uint32 {
	return r.sentPackets
}

// Received returns the number of received packets
func (r *ReportNetwork) Received() uint32 {
	return r.rcvdPackets
}

// Values implements the simul/monitor/CounterIO interface
func (r *ReportNetwork) Values() map[string]float64 {
	return map[string]float64{
		"sent": float64(r.Sent()),
		"rcvd": float64(r.Received()),
	}
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
func (r *ReportStore) Store(sp *incomingSig) *MultiSignature{
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
		"replaceTrial":  float64(r.replacedTrial),
	}
}
