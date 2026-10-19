package main

// Full-sample SIMD benchmarks. The secret key remains in RunFeature's client
// scope; featureServerEvaluate accepts only public coefficients, evaluation keys
// through the evaluator, and encrypted input. Binary serialization exercises
// every client/server boundary, without pretending to measure network latency.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/ring"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type FeatureLDL struct {
	Rows        [][]float64 `json:"rows"`
	Signs       []int       `json:"signs"`
	ComplexReal [][]float64 `json:"complex_real"`
	ComplexImag [][]float64 `json:"complex_imag"`
}

type FeatureModel struct {
	Name     string      `json:"name"`
	Q        [][]float64 `json:"Q"`
	Linear   []float64   `json:"linear"`
	Constant float64     `json:"constant"`
	LDL      FeatureLDL  `json:"ldl"`
}

type featureTerm struct {
	index int
	pt    *rlwe.Plaintext
}

type featurePrepared struct {
	name          string
	rows          [][]featureTerm
	signs         []int
	linear        []featureTerm
	directOffsets []float64
	constant      float64
	encodedBytes  int
	quadraticPM   int
	linearPM      int
	nonlinear     int
	rowRescales   int
	negativeRows  int
}

func featureMust(err error) {
	if err != nil {
		panic(err)
	}
}
func featureSecs(start time.Time) float64   { return time.Since(start).Seconds() }
func featureHash(data []byte) string        { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func featureWrite(path string, data []byte) { featureMust(os.WriteFile(path, data, 0600)) }
func featureHeap() uint64 {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	return memory.HeapAlloc
}
func featureMedian(values []float64) float64 {
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	if len(copyValues) == 0 {
		return 0
	}
	if len(copyValues)%2 == 1 {
		return copyValues[len(copyValues)/2]
	}
	return (copyValues[len(copyValues)/2-1] + copyValues[len(copyValues)/2]) / 2
}

func featurePrepare(params ckks.Parameters, encoder *ckks.Encoder, m FeatureModel, method string) featurePrepared {
	d, slots := len(m.Q), params.MaxSlots()
	p := featurePrepared{name: method, constant: m.Constant}
	encode := func(value complex128, level int, scale rlwe.Scale) *rlwe.Plaintext {
		pt := ckks.NewPlaintext(params, level)
		pt.Scale = scale
		values := make([]complex128, slots)
		for i := range values {
			values[i] = value
		}
		featureMust(encoder.Encode(values, pt))
		p.encodedBytes += pt.BinarySize()
		return pt
	}
	rowScale := rlwe.NewScale(params.Q()[params.MaxLevel()])
	if method == "triangular_direct_q" {
		p.rows = make([][]featureTerm, d)
		p.directOffsets = append([]float64(nil), m.Linear...)
		for i := 0; i < d; i++ {
			for j := i; j < d; j++ {
				coefficient := m.Q[i][j]
				if j > i {
					coefficient *= 2
				}
				if coefficient != 0 {
					p.rows[i] = append(p.rows[i], featureTerm{j, encode(complex(coefficient, 0), params.MaxLevel(), rowScale)})
					p.quadraticPM++
				}
			}
			if len(p.rows[i]) != 0 {
				p.rowRescales++
			}
			if len(p.rows[i]) != 0 || m.Linear[i] != 0 {
				p.nonlinear++
			}
		}
		return p
	}
	var realRows, imagRows [][]float64
	if method == "real_signed_ldl" {
		realRows = m.LDL.Rows
		p.signs = append([]int(nil), m.LDL.Signs...)
	} else if method == "complex_paired_ldl" {
		realRows, imagRows = m.LDL.ComplexReal, m.LDL.ComplexImag
		p.signs = make([]int, len(realRows))
		for i := range p.signs {
			p.signs[i] = 1
		}
	} else {
		panic("unknown feature method: " + method)
	}
	p.rows = make([][]featureTerm, len(realRows))
	for i, row := range realRows {
		if len(row) != d {
			panic("feature LDL dimension mismatch")
		}
		for j, realCoefficient := range row {
			imagCoefficient := 0.0
			if imagRows != nil {
				imagCoefficient = imagRows[i][j]
			}
			coefficient := complex(realCoefficient, imagCoefficient)
			if coefficient != 0 {
				p.rows[i] = append(p.rows[i], featureTerm{j, encode(coefficient, params.MaxLevel(), rowScale)})
				p.quadraticPM++
			}
		}
		if len(p.rows[i]) != 0 {
			p.rowRescales++
			p.nonlinear++
		}
		if p.signs[i] < 0 {
			p.negativeRows++
		}
	}
	// Quadratic products have scale Delta^2 at level 1 before the final
	// rescale. Encode the common linear form at level 1 and scale Delta too.
	for j, coefficient := range m.Linear {
		if coefficient != 0 {
			p.linear = append(p.linear, featureTerm{j, encode(complex(coefficient, 0), params.MaxLevel()-1, params.DefaultScale())})
			p.linearPM++
		}
	}
	return p
}

// featureLinearForm never decrypts and performs no slot movement.
func featureLinearForm(eval *ckks.Evaluator, inputs []*rlwe.Ciphertext, terms []featureTerm) *rlwe.Ciphertext {
	var sum *rlwe.Ciphertext
	for _, term := range terms {
		value, err := eval.MulNew(inputs[term.index], term.pt)
		featureMust(err)
		if sum == nil {
			sum = value
		} else {
			featureMust(eval.Add(sum, value, sum))
		}
	}
	return sum
}

func featureServerEvaluate(params ckks.Parameters, eval *ckks.Evaluator, p featurePrepared, inputs []*rlwe.Ciphertext) *rlwe.Ciphertext {
	// Level changes copy ciphertexts explicitly; their cost remains timed.
	lowerInputs := make([]*rlwe.Ciphertext, len(inputs))
	for i, input := range inputs {
		lowerInputs[i] = eval.DropLevelNew(input, 1)
	}
	var sum *rlwe.Ciphertext
	for i, terms := range p.rows {
		if len(terms) == 0 && (p.directOffsets == nil || p.directOffsets[i] == 0) {
			continue
		}
		z := featureLinearForm(eval, inputs, terms)
		if z == nil {
			var err error
			z, err = eval.MulNew(lowerInputs[i], 0)
			featureMust(err)
		} else {
			featureMust(eval.Rescale(z, z))
		}
		var product *rlwe.Ciphertext
		var err error
		if p.directOffsets != nil {
			featureMust(eval.Add(z, p.directOffsets[i], z))
			product, err = eval.MulNew(lowerInputs[i], z)
		} else {
			product, err = eval.MulNew(z, z)
		}
		featureMust(err)
		if p.directOffsets == nil && p.signs[i] < 0 {
			// Gaussian-integer negation has scale one; no hidden rescale.
			featureMust(eval.Mul(product, -1, product))
		}
		if sum == nil {
			sum = product
		} else {
			featureMust(eval.Add(sum, product, sum))
		}
	}
	if len(p.linear) > 0 {
		linear := featureLinearForm(eval, lowerInputs, p.linear)
		if sum == nil {
			sum = linear
		} else {
			featureMust(eval.Add(sum, linear, sum))
		}
	}
	if sum == nil {
		var err error
		sum, err = eval.MulNew(lowerInputs[0], 0)
		featureMust(err)
		eval.DropLevel(sum, sum.Level())
	} else {
		if sum.Degree() == 2 {
			var err error
			sum, err = eval.RelinearizeNew(sum)
			featureMust(err)
		}
		featureMust(eval.Rescale(sum, sum))
	}
	featureMust(eval.Add(sum, p.constant, sum))
	return sum
}

// RunFeature evaluates three predetermined full-SIMD competitors under the
// same Standard-CKKS parameters. Use RunPackedWithRing for the real-only ring
// challenge; this function deliberately does not silently change ring type.
func RunFeature(m FeatureModel, batch, repeats int, artifactDir string) map[string]any {
	oldProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldProcs)
	start := time.Now()
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN: 13, LogQ: []int{58, 40, 40}, LogP: []int{60}, LogDefaultScale: 40, RingType: ring.Standard,
	})
	featureMust(err)
	parameterSeconds := featureSecs(start)
	d, slots := len(m.Q), params.MaxSlots()
	if batch <= 0 {
		batch = slots
	}
	if d == 0 || batch > slots || batch < 1 || len(m.Linear) != d {
		panic("invalid feature model or batch")
	}
	if repeats < 1 {
		repeats = 1
	}
	if !featureFinite(m.Constant) {
		panic("nonfinite feature constant")
	}
	for _, coefficient := range m.Linear {
		if !featureFinite(coefficient) {
			panic("nonfinite feature linear coefficient")
		}
	}
	for _, row := range m.Q {
		if len(row) != d {
			panic("feature Q must be square")
		}
	}
	for i, row := range m.Q {
		for j, coefficient := range row {
			if !featureFinite(coefficient) || math.Abs(coefficient-m.Q[j][i]) > 1e-13 {
				panic("feature Q must be finite symmetric")
			}
		}
	}
	if len(m.LDL.Rows) != len(m.LDL.Signs) || len(m.LDL.ComplexReal) != len(m.LDL.ComplexImag) {
		panic("invalid feature LDL shapes")
	}
	for _, sign := range m.LDL.Signs {
		if sign != 1 && sign != -1 {
			panic("feature LDL signs must be +1 or -1")
		}
	}
	for _, rows := range [][][]float64{m.LDL.Rows, m.LDL.ComplexReal, m.LDL.ComplexImag} {
		for _, row := range rows {
			if len(row) != d {
				panic("invalid feature LDL row dimension")
			}
			for _, coefficient := range row {
				if !featureFinite(coefficient) {
					panic("nonfinite feature LDL coefficient")
				}
			}
		}
	}
	featureMust(os.MkdirAll(artifactDir, 0700))
	rng := rand.New(rand.NewSource(20261019 + int64(d)))
	inputs, reference := make([][]float64, batch), make([]float64, batch)
	for b := 0; b < batch; b++ {
		inputs[b] = make([]float64, d)
		for i := 0; i < d; i++ {
			x := float64(rng.Intn(2049)-1024) / 1024
			switch b {
			case 0:
				x = 0
			case 1:
				x = 1
			case 2:
				x = -1
			case 3:
				if i%2 == 0 {
					x = 1
				} else {
					x = -1
				}
			case 4:
				if i == 0 {
					x = 1
				} else {
					x = 0
				}
			}
			inputs[b][i] = x
		}
		y := m.Constant
		for i := 0; i < d; i++ {
			y += m.Linear[i] * inputs[b][i]
			for j := 0; j < d; j++ {
				y += m.Q[i][j] * inputs[b][i] * inputs[b][j]
			}
		}
		reference[b] = y
	}
	fixture, err := json.Marshal(map[string]any{"model": m, "inputs": inputs, "reference": reference})
	featureMust(err)
	featureWrite(filepath.Join(artifactDir, "feature_fixture.json"), fixture)
	start = time.Now()
	kgen := rlwe.NewKeyGenerator(params)
	sk, pk := kgen.GenKeyPairNew()
	keypairSeconds := featureSecs(start)
	start = time.Now()
	rlk := kgen.GenRelinearizationKeyNew(sk)
	keygenSeconds := featureSecs(start)
	start = time.Now()
	pkBytes, err := pk.MarshalBinary()
	featureMust(err)
	pkSerializeSeconds := featureSecs(start)
	start = time.Now()
	clientPK := new(rlwe.PublicKey)
	featureMust(clientPK.UnmarshalBinary(pkBytes))
	pkDeserializeSeconds := featureSecs(start)
	start = time.Now()
	evkBytes, err := rlwe.NewMemEvaluationKeySet(rlk).MarshalBinary()
	featureMust(err)
	keySerializeSeconds := featureSecs(start)
	start = time.Now()
	serverKeys := new(rlwe.MemEvaluationKeySet)
	featureMust(serverKeys.UnmarshalBinary(evkBytes))
	keyDeserializeSeconds := featureSecs(start)
	featureWrite(filepath.Join(artifactDir, "feature_public_key.bin"), pkBytes)
	featureWrite(filepath.Join(artifactDir, "feature_evaluation_keys.bin"), evkBytes)
	encoder := ckks.NewEncoder(params)
	encryptor, decryptor := rlwe.NewEncryptor(params, clientPK), rlwe.NewDecryptor(params, sk)
	evaluator := ckks.NewEvaluator(params, serverKeys)
	methods := []string{"triangular_direct_q", "real_signed_ldl", "complex_paired_ldl"}
	if selected := os.Getenv("FHE_FEATURE_METHOD"); selected != "" {
		found := false
		for _, method := range methods {
			if method == selected {
				found = true
			}
		}
		if !found {
			panic("unknown FHE_FEATURE_METHOD")
		}
		methods = []string{selected}
	}
	results := make([]map[string]any, 0, len(methods))
	for _, method := range methods {
		runtime.GC()
		heapBefore, heapMax := featureHeap(), featureHeap()
		capture := func() {
			if value := featureHeap(); value > heapMax {
				heapMax = value
			}
		}
		start = time.Now()
		prepared := featurePrepare(params, encoder, m, method)
		prepareSeconds := featureSecs(start)
		capture()
		samples := map[string][]float64{}
		record := func(name string, started time.Time) { samples[name] = append(samples[name], featureSecs(started)) }
		clientEncode := func() []*rlwe.Plaintext {
			plaintexts := make([]*rlwe.Plaintext, d)
			for i := 0; i < d; i++ {
				values := make([]float64, slots)
				for s := range values {
					values[s] = inputs[s%batch][i]
				}
				plaintexts[i] = ckks.NewPlaintext(params, params.MaxLevel())
				featureMust(encoder.Encode(values, plaintexts[i]))
			}
			return plaintexts
		}
		clientEncrypt := func(plaintexts []*rlwe.Plaintext) []*rlwe.Ciphertext {
			ciphertexts := make([]*rlwe.Ciphertext, len(plaintexts))
			for i, plaintext := range plaintexts {
				var e error
				ciphertexts[i], e = encryptor.EncryptNew(plaintext)
				featureMust(e)
			}
			return ciphertexts
		}
		start = time.Now()
		warmInputs := clientEncrypt(clientEncode())
		warmClientSeconds := featureSecs(start)
		start = time.Now()
		_ = featureServerEvaluate(params, evaluator, prepared, warmInputs)
		warmServerSeconds := featureSecs(start)
		warmInputs = nil
		maxAbs, maxImag, maxImagError, sumSquared := 0.0, 0.0, 0.0, 0.0
		var outputBytes []byte
		var inputWires [][]byte
		var lastDecoded []float64
		outputLevel, outputScale := 0, 0.0
		for rep := 0; rep < repeats; rep++ {
			chainStart := time.Now()
			start = time.Now()
			plaintexts := clientEncode()
			record("client_pack_encode_seconds", start)
			start = time.Now()
			clientInputs := clientEncrypt(plaintexts)
			record("client_encrypt_seconds", start)
			start = time.Now()
			inputWires = make([][]byte, d)
			for i, ciphertext := range clientInputs {
				inputWires[i], err = ciphertext.MarshalBinary()
				featureMust(err)
			}
			record("input_serialize_seconds", start)
			start = time.Now()
			serverInputs := make([]*rlwe.Ciphertext, d)
			for i, wire := range inputWires {
				serverInputs[i] = new(rlwe.Ciphertext)
				featureMust(serverInputs[i].UnmarshalBinary(wire))
			}
			record("input_deserialize_seconds", start)
			start = time.Now()
			serverOutput := featureServerEvaluate(params, evaluator, prepared, serverInputs)
			record("server_compute_seconds", start)
			outputLevel, outputScale = serverOutput.Level(), serverOutput.Scale.Float64()
			start = time.Now()
			outputBytes, err = serverOutput.MarshalBinary()
			featureMust(err)
			record("output_serialize_seconds", start)
			start = time.Now()
			clientOutput := new(rlwe.Ciphertext)
			featureMust(clientOutput.UnmarshalBinary(outputBytes))
			record("output_deserialize_seconds", start)
			start = time.Now()
			decoded := make([]complex128, slots)
			featureMust(encoder.Decode(decryptor.DecryptNew(clientOutput), decoded))
			record("client_decrypt_decode_seconds", start)
			record("complete_local_chain_seconds", chainStart)
			capture()
			lastDecoded = make([]float64, batch)
			for s, value := range decoded {
				if math.IsNaN(real(value)) || math.IsInf(real(value), 0) || math.IsNaN(imag(value)) || math.IsInf(imag(value), 0) {
					panic("nonfinite feature output")
				}
				b := s % batch
				errorAbs := math.Abs(real(value) - reference[b])
				if errorAbs > maxAbs {
					maxAbs = errorAbs
				}
				if imaginary := math.Abs(imag(value)); imaginary > maxImag {
					maxImag = imaginary
				}
				expectedImag := 0.0
				if method == "complex_paired_ldl" {
					for k, row := range m.LDL.ComplexReal {
						re, im := 0.0, 0.0
						for j, coefficient := range row {
							re += coefficient * inputs[b][j]
							im += m.LDL.ComplexImag[k][j] * inputs[b][j]
						}
						expectedImag += 2 * re * im
					}
				}
				if imaginaryError := math.Abs(imag(value) - expectedImag); imaginaryError > maxImagError {
					maxImagError = imaginaryError
				}
				sumSquared += errorAbs * errorAbs
				if s < batch {
					lastDecoded[s] = real(value)
				}
			}
			if rep == repeats-1 {
				for i, wire := range inputWires {
					featureWrite(filepath.Join(artifactDir, fmt.Sprintf("%s_input_%02d.ct", method, i)), wire)
				}
				featureWrite(filepath.Join(artifactDir, method+"_output.ct"), outputBytes)
			}
		}
		if maxAbs > 1e-3 {
			panic(fmt.Sprintf("feature quality gate failed: %s %s %.17g", m.Name, method, maxAbs))
		}
		medians := map[string]float64{}
		for name, values := range samples {
			medians[name] = featureMedian(values)
		}
		inputByteCount, hashes := 0, make([]string, len(inputWires))
		for i, wire := range inputWires {
			inputByteCount += len(wire)
			hashes[i] = featureHash(wire)
		}
		relinearizations := 0
		if prepared.nonlinear > 0 {
			relinearizations = 1
		}
		result := map[string]any{
			"method": method, "quality_passed": true, "quadratic_plaintext_multiplications": prepared.quadraticPM, "linear_plaintext_multiplications": prepared.linearPM,
			"plaintext_multiplications": prepared.quadraticPM + prepared.linearPM, "ciphertext_multiplications": prepared.nonlinear,
			"ciphertext_squares": func() int {
				if method == "triangular_direct_q" {
					return 0
				}
				return prepared.nonlinear
			}(),
			"cheap_sign_negations": prepared.negativeRows, "relinearizations": relinearizations, "rescalings": prepared.rowRescales + 1,
			"rotations": 0, "rotation_key_count": 0, "conjugations": 0, "bootstraps": 0, "variable_variable_depth": 1, "rescale_levels_consumed": 2,
			"input_level": params.MaxLevel(), "output_level": outputLevel, "input_scale": params.DefaultScale().Float64(), "output_scale": outputScale,
			"coefficient_policy":    "preencoded constant-slot plaintexts; quadratic coefficient scale Q[last], linear coefficient scale default Delta; zeros skipped",
			"linear_term_policy":    "direct Q folds linear_i into each pre-product row; LDL adds a shared linear form at degree-two scale before final rescale",
			"model_prepare_seconds": prepareSeconds, "encoded_public_model_bytes": prepared.encodedBytes,
			"warmup_client_encode_encrypt_seconds": warmClientSeconds, "warmup_server_seconds": warmServerSeconds,
			"input_ciphertext_count": d, "input_ciphertext_bytes": inputByteCount, "output_ciphertext_bytes": len(outputBytes),
			"evaluation_key_serialized_bytes": len(evkBytes), "heap_before_method_bytes": heapBefore, "heap_max_stage_snapshot_bytes": heapMax,
			"heap_measurement": "stage snapshots of Go HeapAlloc; not process peak RSS", "samples": samples, "median": medians,
			"max_abs_error_all_lanes_repeats": maxAbs, "rmse": math.Sqrt(sumSquared / float64(slots*repeats)),
			"max_abs_imaginary_all_lanes_repeats": maxImag, "max_abs_imaginary_error_from_compiled_model": maxImagError,
			"imaginary_policy":   "complex method's intentional cross terms are discarded by client only after final decryption",
			"decoded_first_lane": lastDecoded, "input_sha256_last_repeat": hashes, "output_sha256_last_repeat": featureHash(outputBytes),
		}
		data, e := json.MarshalIndent(result, "", "  ")
		featureMust(e)
		featureWrite(filepath.Join(artifactDir, method+".json"), data)
		results = append(results, result)
		fmt.Printf("feature %s %s d=%d batch=%d server=%.6fs chain=%.6fs error=%.3g\n", m.Name, method, d, batch, medians["server_compute_seconds"], medians["complete_local_chain_seconds"], maxAbs)
	}
	return map[string]any{
		"model": m.Name, "dimension": d, "batch": batch, "physical_slots": slots, "repeats": repeats, "fixture_sha256": featureHash(fixture),
		"parameter_construction_seconds": parameterSeconds, "client_keypair_generation_seconds": keypairSeconds,
		"evaluation_key_generation_seconds": keygenSeconds, "public_key_serialize_seconds": pkSerializeSeconds, "public_key_deserialize_seconds": pkDeserializeSeconds,
		"key_serialize_seconds": keySerializeSeconds, "key_deserialize_seconds": keyDeserializeSeconds,
		"public_key_bytes": len(pkBytes), "evaluation_key_serialized_bytes": len(evkBytes), "key_serialized_sha256": featureHash(evkBytes),
		"logN": params.LogN(), "ring_type": params.RingType().String(), "Q": params.Q(), "P": params.P(), "logQP": params.LogQP(),
		"log_default_scale": params.LogDefaultScale(), "thread_count": 1, "layout": "one ciphertext per input feature; samples in slots; periodic sample repetition if batch<slots",
		"transport":   "actual binary serialization and deserialization in one process; physical network latency unmeasured",
		"artifact_io": "fixture/key/ciphertext evidence writes outside timed setup/inference", "results": results,
	}
}

func featureFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
