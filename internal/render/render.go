// Package render превращает []parse.Rule в текст формата Prometheus exposition.
package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dregor/iptables-comment-exporter/internal/parse"
)

// Render формирует полный текст ответа /metrics по списку правил.
func Render(rules []parse.Rule) string {
	var b strings.Builder

	b.WriteString("# HELP iptables_comment_rule_packets_total Пакеты, посчитанные помеченным правилом iptables (iptables-save -c).\n")
	b.WriteString("# TYPE iptables_comment_rule_packets_total counter\n")
	for _, r := range rules {
		fmt.Fprintf(&b, "iptables_comment_rule_packets_total%s %d\n", labels(r), r.Packets)
	}

	b.WriteString("# HELP iptables_comment_rule_bytes_total Байты, посчитанные помеченным правилом iptables (iptables-save -c).\n")
	b.WriteString("# TYPE iptables_comment_rule_bytes_total counter\n")
	for _, r := range rules {
		fmt.Fprintf(&b, "iptables_comment_rule_bytes_total%s %d\n", labels(r), r.Bytes)
	}

	return b.String()
}

func labels(r parse.Rule) string {
	return fmt.Sprintf(
		"{family=%s,table=%s,chain=%s,label=%s}",
		strconv.Quote(r.Family), strconv.Quote(r.Table), strconv.Quote(r.Chain), strconv.Quote(r.Label),
	)
}
