package handel

// ReportHandel holds a handel struct but modifies it so it is able to issue
// some stats.
type ReportHandel struct {
	*Handel
}

// Stats contains different stats about the different components of Handel. Not
// complete.
type Stats struct {
	Network map[string]float64
	Store   map[string]float64
}

// NewReportHandel returns a Handel that can report some statistis about its
// internals
func NewReportHandel(h *Handel) *ReportHandel {
	h.net = NewReportNetwork(h.net)
	h.store = newReportStore(h.store)
	return &ReportHandel{h}
}

// Stats returns the stats of internal components of Handel.
func (r *ReportHandel) Stats() *Stats {
	s := new(Stats)
	net := r.Handel.net.(*ReportNetwork)
	s.Network = net.Values()
	store := r.Handel.store.(*ReportStore)
	s.Store = store.Values()
	return s
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
		"sentPackets": float64(r.Sent()),
		"rcvdPackets": float64(r.Received()),
	}
}

// ReportStore is a Store that can report some statistics about the storage
type ReportStore struct {
	signatureStore
	sucessReplaced int64
}

// newReportStore returns a signatureStore with som eadditional reporting
// capabilities
func newReportStore(s signatureStore) signatureStore {
	return &ReportStore{
		signatureStore: s,
	}
}

// Store implements the signatureStore interface
func (r *ReportStore) Store(level byte, ms *MultiSignature) (*MultiSignature, bool) {
	ms, isNew := r.signatureStore.Store(level, ms)
	if isNew {
		r.sucessReplaced++
	}
	return ms, isNew
}

// Values implements the simul/monitor/counterIO interface
func (r *ReportStore) Values() map[string]float64 {
	return map[string]float64{
		"sucessReplace": float64(r.sucessReplaced),
	}
}
