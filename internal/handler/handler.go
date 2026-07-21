// Package handler отдаёт HTTP-хендлер для /metrics: на каждый скрейп заново
// вызывает *tables-save -c и превращает результат в текст Prometheus.
package handler

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dregor/iptables-comment-exporter/internal/collect"
	"github.com/dregor/iptables-comment-exporter/internal/parse"
	"github.com/dregor/iptables-comment-exporter/internal/render"
)

// Config — параметры сбора метрик.
type Config struct {
	Prefix  string        // обязательный префикс --comment
	Timeout time.Duration // таймаут одного вызова *tables-save
	NoIPv6  bool          // не собирать ip6tables-save
}

// Metrics возвращает http.Handler для /metrics. Скрейпы сериализуются мьютексом,
// чтобы параллельные запросы не плодили несколько *tables-save одновременно.
func Metrics(cfg Config) http.Handler {
	var mu sync.Mutex

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		ctx, cancel := context.WithTimeout(r.Context(), cfg.Timeout*2+time.Second)
		defer cancel()

		var rules []parse.Rule
		rules = append(rules, collectFamily(ctx, "iptables-save", "ipv4", cfg)...)
		if !cfg.NoIPv6 {
			rules = append(rules, collectFamily(ctx, "ip6tables-save", "ipv6", cfg)...)
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if _, err := w.Write([]byte(render.Render(rules))); err != nil {
			log.Printf("write response: %v", err)
		}
	})
}

func collectFamily(ctx context.Context, bin, family string, cfg Config) []parse.Rule {
	data, err := collect.Run(ctx, bin, cfg.Timeout)
	if err != nil {
		log.Printf("%s: %v", bin, err)
		return nil
	}
	return parse.Parse(strings.NewReader(string(data)), family, cfg.Prefix)
}
