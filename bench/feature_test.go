package main

import (
	"encoding/json"
	"os"
	"testing"
)

// This is a real encrypted execution test, including the serialized transport
// boundary. It checks a frozen model instead of mirroring the implementation.
func TestFeatureEncryptedSmoke(t *testing.T) {
	data, err := os.ReadFile("../artifacts/models.json")
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		Models []FeatureModel `json:"models"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Models) == 0 {
		t.Fatal("no frozen models")
	}
	result := RunFeature(registry.Models[0], 4096, 1, t.TempDir())
	for _, method := range result["results"].([]map[string]any) {
		if err := method["max_abs_error_all_lanes_repeats"].(float64); err > 1e-4 {
			t.Fatalf("%s error %.8g exceeds smoke tolerance", method["method"], err)
		}
		if method["output_level"].(int) != 0 {
			t.Fatalf("unexpected level: %v", method)
		}
	}
}
