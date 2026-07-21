// Package parse разбирает вывод "iptables-save -c" / "ip6tables-save -c"
// и оставляет только правила с меткой в --comment.
package parse

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// maxLine — предел длины одной строки правила (реальные правила короче,
// это защита от аномально длинного/битого ввода).
const maxLine = 1 << 20 // 1MiB

var (
	counterLineRe = regexp.MustCompile(`^\[(\d+):(\d+)\]\s+-A\s+(\S+)`)
	commentRe     = regexp.MustCompile(`--comment\s+"([^"]*)"`)
)

// Parse читает построчно ввод *tables-save и возвращает правила, чей
// --comment начинается с prefix; сама метка — остаток строки после prefix.
func Parse(r io.Reader, family, prefix string) []Rule {
	var rules []Rule
	var table string

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLine)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "*") {
			table = strings.TrimPrefix(line, "*")
			continue
		}

		m := counterLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		cm := commentRe.FindStringSubmatch(line)
		if cm == nil || !strings.HasPrefix(cm[1], prefix) {
			continue
		}

		packets, _ := strconv.ParseUint(m[1], 10, 64)
		bytesCount, _ := strconv.ParseUint(m[2], 10, 64)

		rules = append(rules, Rule{
			Family:  family,
			Table:   table,
			Chain:   m[3],
			Label:   strings.TrimPrefix(cm[1], prefix),
			Packets: packets,
			Bytes:   bytesCount,
		})
	}

	return rules
}
