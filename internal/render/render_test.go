package render

import (
	"strings"
	"testing"

	"github.com/dregor/iptables-comment-exporter/internal/parse"
)

func TestRender(t *testing.T) {
	rules := []parse.Rule{
		{Family: "ipv4", Table: "filter", Chain: "INPUT", Label: "default_last_dropped", Packets: 89, Bytes: 1011},
	}
	out := Render(rules)

	for _, want := range []string{
		`iptables_comment_rule_packets_total{family="ipv4",table="filter",chain="INPUT",label="default_last_dropped"} 89`,
		`iptables_comment_rule_bytes_total{family="ipv4",table="filter",chain="INPUT",label="default_last_dropped"} 1011`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("в выводе не найдена строка: %s\n---\n%s", want, out)
		}
	}
}

func TestRenderEscaping(t *testing.T) {
	rules := []parse.Rule{
		{Family: "ipv4", Table: "filter", Chain: "INPUT", Label: `weird"label\here`, Packets: 1, Bytes: 2},
	}
	out := Render(rules)
	if !strings.Contains(out, `label="weird\"label\\here"`) {
		t.Errorf("метка не экранирована корректно:\n%s", out)
	}
}

func TestRenderEmpty(t *testing.T) {
	out := Render(nil)
	if !strings.Contains(out, "# TYPE iptables_comment_rule_packets_total counter") {
		t.Errorf("даже без правил должны быть заголовки HELP/TYPE:\n%s", out)
	}
}
