// iptables-comment-exporter отдаёт Prometheus-метрики по счётчикам пакетов/байт
// правил iptables, помеченных --comment "<prefix><label>". Смотри README.md.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/dregor/iptables-comment-exporter/internal/handler"
)

func main() {
	bind := flag.String("bind", "127.0.0.1:9631", "адрес:порт для /metrics")
	prefix := flag.String("prefix", "iptables-exporter ", "обязательный префикс --comment; правила без него игнорируются")
	timeout := flag.Duration("timeout", 5*time.Second, "таймаут одного вызова *tables-save за скрейп")
	noIPv6 := flag.Bool("no-ipv6", false, "не собирать ip6tables-save")
	memLimitMB := flag.Int64("mem-limit-mb", 128, "мягкий лимит памяти рантайма Go, МиБ (0 = не ограничивать)")
	flag.Parse()

	runtime.GOMAXPROCS(1) // редкие короткие скрейпы, второй CPU не нужен
	if *memLimitMB > 0 {
		debug.SetMemoryLimit(*memLimitMB << 20)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", handler.Metrics(handler.Config{
		Prefix:  *prefix,
		Timeout: *timeout,
		NoIPv6:  *noIPv6,
	}))

	srv := &http.Server{
		Addr:              *bind,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      *timeout*2 + 5*time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	go func() {
		log.Printf("iptables-comment-exporter слушает %s", *bind)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
