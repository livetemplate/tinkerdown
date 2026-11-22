package server

import "testing"

// TestBlockHelperFuncsReachParseAndTreeGen guards that every one of
// Tinkerdown's base block helpers (blockHelperFuncs) reaches BOTH funcMaps a
// block needs at serve time:
//   - the component set's Funcs, which reach the block's PARSE (New / Validate);
//   - getComponentFuncs, applied via tmpl.Funcs after New for TREE GENERATION.
//
// A helper present in only one would let the block parse and then fail to
// render, or vice versa — the exact split-brain that made split's absence
// invisible until serve.
func TestBlockHelperFuncsReachParseAndTreeGen(t *testing.T) {
	sets := ComponentTemplates()
	if len(sets) == 0 {
		t.Fatal("ComponentTemplates() returned no sets")
	}
	treeFuncs := getComponentFuncs()
	for name := range blockHelperFuncs {
		if sets[0].Funcs[name] == nil {
			t.Errorf("%s missing from ComponentTemplates()[0].Funcs — a block using it would fail to PARSE", name)
		}
		if treeFuncs[name] == nil {
			t.Errorf("%s missing from getComponentFuncs() — a block using it would fail TREE GENERATION", name)
		}
	}
}
