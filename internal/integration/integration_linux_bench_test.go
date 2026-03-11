//go:build linux
// +build linux

package integration_test

import (
	"testing"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/mdlayher/netlink/internal/integration/testutil"
)

func BenchmarkNftablesDump(b *testing.B) {
	testutil.SkipUnprivileged(b)
	sizes := []struct {
		name  string
		rules int
	}{
		{name: "1", rules: 1},
		{name: "8", rules: 8},
		{name: "64", rules: 64},
		{name: "512", rules: 512},
		{name: "4096", rules: 4096},
		{name: "32768", rules: 32768},
	}

	for _, sz := range sizes {
		ns, closeNS := testutil.NewNS(b)
		conn := testutil.NewNftablesConn(b, ns)

		table := &nftables.Table{
			Name:   "bench",
			Family: nftables.TableFamilyIPv4,
		}
		conn.AddTable(table)

		chain := &nftables.Chain{
			Table: table,
			Name:  "bench_chain",
			Type:  nftables.ChainTypeFilter,
		}
		conn.AddChain(chain)

		rules := make([]*nftables.Rule, sz.rules)
		for i := range sz.rules {
			rules[i] = &nftables.Rule{
				Table: table,
				Chain: chain,
				Exprs: []expr.Any{
					&expr.Verdict{
						Kind: expr.VerdictAccept,
					},
				},
			}
			conn.AddRule(rules[i])
		}
		if err := conn.Flush(); err != nil {
			b.Fatalf("failed to flush nftables: %v", err)
		}

		b.Run(sz.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				rules, err := conn.GetRules(table, chain)
				if err != nil {
					b.Fatalf("failed to get rules: %v", err)
				}
				if len(rules) != sz.rules {
					b.Fatalf("expected %d rules, got %d", sz.rules, len(rules))
				}
			}
		})

		closeNS()
	}
}
