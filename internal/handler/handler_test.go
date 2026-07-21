package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsNoIPv6(t *testing.T) {
	// без реальных iptables-save/ip6tables-save в PATH сбор просто вернёт
	// пустой список правил, но хендлер обязан ответить 200 и валидными заголовками.
	h := Metrics(Config{Prefix: "iptables-exporter ", Timeout: time.Second, NoIPv6: true})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("код ответа = %d, ожидался 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, ожидался text/plain/...", ct)
	}
	if !strings.Contains(rec.Body.String(), "iptables_comment_rule_packets_total") {
		t.Errorf("в ответе должны быть заголовки HELP/TYPE даже без правил:\n%s", rec.Body.String())
	}
}
