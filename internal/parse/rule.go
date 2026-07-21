package parse

// Rule — одно правило iptables/ip6tables со счётчиками и меткой,
// извлечённой из --comment.
type Rule struct {
	Family  string // ipv4 / ipv6
	Table   string // filter, nat, ...
	Chain   string // INPUT, PREROUTING, ...
	Label   string // текст после prefix в --comment
	Packets uint64
	Bytes   uint64
}
