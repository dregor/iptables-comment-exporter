// Package collect запускает *tables-save и отдаёт его вывод с ограничением
// по времени и по объёму, чтобы зависший или аномальный бинарник не повесил
// и не раздул процесс.
package collect

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// MaxOutput — верхняя граница объёма вывода *tables-save.
const MaxOutput = 32 << 20 // 32MiB

// Run выполняет "<bin> -c" с таймаутом timeout и возвращает stdout.
func Run(ctx context.Context, bin string, timeout time.Duration) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, bin, "-c")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("%s: stdout pipe: %w", bin, err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%s: start: %w", bin, err)
	}

	data, readErr := io.ReadAll(io.LimitReader(stdout, MaxOutput+1))
	waitErr := cmd.Wait()

	if readErr != nil {
		return nil, fmt.Errorf("%s: read: %w", bin, readErr)
	}
	if waitErr != nil {
		return nil, fmt.Errorf("%s: %w", bin, waitErr)
	}
	if len(data) > MaxOutput {
		return nil, fmt.Errorf("%s: вывод превышает лимит %d байт", bin, MaxOutput)
	}
	return data, nil
}
