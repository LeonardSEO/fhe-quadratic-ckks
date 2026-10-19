package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func main() {
	if MaybeRunServerProcess() {
		return
	}
	modelsPath := flag.String("models", "../artifacts/models.json", "frozen model file")
	selected := flag.String("model", "all", "model name or all")
	out := flag.String("out", "../artifacts/runs", "output directory")
	repeats := flag.Int("repeats", 9, "timed repetitions after warmup")
	batch := flag.Int("batch", 128, "fixed batch, 0 selects full standard slots/d")
	ci := flag.Bool("ci", false, "conjugate invariant CKKS")
	mode := flag.String("mode", "packed", "packed, feature, factors, or network")
	flag.Parse()
	runtime.GOMAXPROCS(1)
	data, err := os.ReadFile(*modelsPath)
	packedMust(err)
	var frozen struct {
		Models []PackedModel `json:"models"`
	}
	packedMust(json.Unmarshal(data, &frozen))
	packedMust(os.MkdirAll(*out, 0700))
	outputs := []map[string]any{}
	found := false
	for _, m := range frozen.Models {
		if *selected != "all" && *selected != m.Name {
			continue
		}
		found = true
		ringName := "standard"
		if *ci {
			ringName = "ci"
		}
		dest := filepath.Join(*out, m.Name, ringName)
		var r map[string]any
		switch *mode {
		case "packed":
			r = RunPackedWithRing(m, *batch, *repeats, dest, *ci)
		case "feature":
			var full struct {
				Models []FeatureModel `json:"models"`
			}
			packedMust(json.Unmarshal(data, &full))
			for _, fm := range full.Models {
				if fm.Name == m.Name {
					r = RunFeature(fm, *batch, *repeats, dest)
				}
			}
		case "factors":
			var full struct {
				Models []FactorPackedModel `json:"models"`
			}
			packedMust(json.Unmarshal(data, &full))
			for _, fm := range full.Models {
				if fm.Name == m.Name {
					r = RunFactorPacked(fm, *batch, *repeats, dest)
				}
			}
		case "network":
			var full struct {
				Models []NetworkModel `json:"models"`
			}
			packedMust(json.Unmarshal(data, &full))
			for _, fm := range full.Models {
				if fm.Name == m.Name {
					r = RunNetwork(fm, *batch, *repeats, dest)
				}
			}
		default:
			panic("unknown mode")
		}
		outputs = append(outputs, r)
		encoded, err := json.MarshalIndent(r, "", "  ")
		packedMust(err)
		packedWrite(filepath.Join(dest, "run.json"), encoded)
	}
	if !found {
		panic("model not found")
	}
	encoded, err := json.MarshalIndent(map[string]any{"recorded_at": time.Now().UTC().Format(time.RFC3339), "go_version": runtime.Version(), "GOOS": runtime.GOOS, "GOARCH": runtime.GOARCH, "GOMAXPROCS": 1, "models_sha256": packedHash(data), "runs": outputs}, "", "  ")
	packedMust(err)
	packedWrite(filepath.Join(*out, "aggregate.json"), encoded)
	fmt.Printf("wrote %s\n", filepath.Join(*out, "aggregate.json"))
}
