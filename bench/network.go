package main

// Original public-weight square network baseline. Hidden neurons are SIMD
// packed in chunks; the evaluator sees no secret key and performs no decryption.
import (
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/tuneinsight/lattigo/v6/circuits/ckks/lintrans"
	commonlt "github.com/tuneinsight/lattigo/v6/circuits/common/lintrans"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/ring"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type NetworkModel struct {
	Name          string      `json:"name"`
	W             [][]float64 `json:"W"`
	V             []float64   `json:"v"`
	Bias          []float64   `json:"bias"`
	OutputBias    float64     `json:"output_bias"`
	Normalization float64     `json:"normalization"`
	Q             [][]float64 `json:"Q"`
	Linear        []float64   `json:"linear"`
	Constant      float64     `json:"constant"`
}

type networkPrepared struct {
	transforms     []lintrans.LinearTransformation
	biases         []*rlwe.Plaintext
	parameters     []lintrans.Parameters
	encodedBytes   int
	transformBytes int
	biasBytes      int
}

// Input lane b, feature j lives at j*batch+b, repeated every d*batch.
// In each output chunk all slots/batch rows are consumed, with unused neurons
// explicitly zero. sqrt(v_i*normalization) makes the sum of complex squares
// equal to the original signed real network without a post-square multiply.
func networkPrepare(params ckks.Parameters, encoder *ckks.Encoder, m NetworkModel, batch, ratio int) networkPrepared {
	slots, d := params.MaxSlots(), len(m.W[0])
	rows := slots / batch
	chunks := (len(m.W) + rows - 1) / rows
	p := networkPrepared{}
	foldedW := make([][]complex128, len(m.W))
	foldedBias := make([]complex128, len(m.W))
	for neuron, row := range m.W {
		weight := cmplx.Sqrt(complex(m.Normalization*m.V[neuron], 0))
		foldedW[neuron] = make([]complex128, d)
		for j, value := range row {
			foldedW[neuron][j] = weight * complex(value, 0)
		}
		foldedBias[neuron] = weight * complex(m.Bias[neuron], 0)
	}
	inputPT := ckks.NewPlaintext(params, params.MaxLevel())
	for chunk := 0; chunk < chunks; chunk++ {
		diagonals := make(lintrans.Diagonals[complex128])
		for r := 0; r < d; r++ {
			values := make([]complex128, slots)
			nonzero := false
			for s := range values {
				row := s / batch
				neuron := chunk*rows + row
				if neuron < len(m.W) {
					values[s] = foldedW[neuron][(row+r)%d]
					nonzero = nonzero || values[s] != 0
				}
			}
			if nonzero {
				diagonals[r*batch] = values
			}
		}
		// The frozen models have nonzero W. A pure constant chunk would otherwise
		// have an empty transform, unsupported by the library's diagonal evaluator.
		if len(diagonals) == 0 {
			diagonals[0] = make([]complex128, slots)
		}
		ltp := lintrans.Parameters{DiagonalsIndexList: diagonals.DiagonalsIndexList(), LevelQ: params.MaxLevel(), LevelP: params.MaxLevelP(), Scale: rlwe.NewScale(params.Q()[params.MaxLevel()]), LogDimensions: inputPT.LogDimensions, LogBabyStepGiantStepRatio: ratio}
		lt := lintrans.NewTransformation(params, ltp)
		packedMust(lintrans.Encode(encoder, diagonals, lt))
		bias := ckks.NewPlaintext(params, params.MaxLevel()-1)
		values := make([]complex128, slots)
		for s := range values {
			neuron := chunk*rows + s/batch
			if neuron < len(m.W) {
				values[s] = foldedBias[neuron]
			}
		}
		packedMust(encoder.Encode(values, bias))
		for _, v := range lt.Vec {
			p.transformBytes += v.BinarySize()
		}
		p.biasBytes += bias.BinarySize()
		p.encodedBytes = p.transformBytes + p.biasBytes
		p.transforms = append(p.transforms, lt)
		p.biases = append(p.biases, bias)
		p.parameters = append(p.parameters, ltp)
	}
	return p
}

// Official EvaluateManyNew shares the input decomposition, and reuses BSGS
// baby rotations between chunks. Products remain degree two until their sum;
// this costs one final relinearization, rather than one per hidden chunk.
func networkServerEvaluate(params ckks.Parameters, eval *ckks.Evaluator, p networkPrepared, input *rlwe.Ciphertext, batch int, constant float64) *rlwe.Ciphertext {
	chunks, err := lintrans.NewEvaluator(eval).EvaluateManyNew(input, p.transforms)
	packedMust(err)
	var sum *rlwe.Ciphertext
	for i, z := range chunks {
		packedMust(eval.Rescale(z, z))
		packedMust(eval.Add(z, p.biases[i], z))
		square, err := eval.MulNew(z, z)
		packedMust(err)
		if sum == nil {
			sum = square
		} else {
			packedMust(eval.Add(sum, square, sum))
		}
	}
	packedMust(eval.Relinearize(sum, sum))
	packedMust(eval.Rescale(sum, sum))
	for step := 1; step < params.MaxSlots()/batch; step <<= 1 {
		rotated, err := eval.RotateNew(sum, step*batch)
		packedMust(err)
		packedMust(eval.Add(sum, rotated, sum))
	}
	packedMust(eval.Add(sum, constant, sum))
	return sum
}

func networkFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// RunNetwork benchmarks the original network, holding the primary packed
// quadratic harness's parameters, batch, dyadic inputs and quality gate fixed.
// All timing samples include fresh randomized encryption; wire boundaries use
// real binary serialization, while network latency is explicitly unmeasured.
func RunNetwork(m NetworkModel, batch, repeats int, artifactDir string) map[string]any {
	oldProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldProcs)
	start := time.Now()
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{LogN: 13, LogQ: []int{58, 40, 40}, LogP: []int{60}, LogDefaultScale: 40, RingType: ring.Standard})
	packedMust(err)
	parameterSeconds := packedSecs(start)
	if len(m.W) == 0 || len(m.W[0]) == 0 {
		panic("empty network")
	}
	d, hidden, slots := len(m.W[0]), len(m.W), params.MaxSlots()
	if d < 2 || d&(d-1) != 0 || slots%d != 0 {
		panic("network input dimension must be a power of two >=2 dividing slots")
	}
	if len(m.V) != hidden || len(m.Bias) != hidden || !networkFinite(m.Normalization) || !networkFinite(m.OutputBias) || m.Normalization <= 0 {
		panic("network coefficient dimensions or normalization invalid")
	}
	for i, row := range m.W {
		if len(row) != d || !networkFinite(m.V[i]) || !networkFinite(m.Bias[i]) {
			panic("invalid network row")
		}
		for _, value := range row {
			if !networkFinite(value) {
				panic("nonfinite network weight")
			}
		}
	}
	if batch <= 0 {
		batch = slots / d
	}
	if batch*d > slots || slots%(batch*d) != 0 {
		panic("network batch must permit exact periodic feature replication")
	}
	rows := slots / batch
	if rows&(rows-1) != 0 {
		panic("network chunk row count must be a power of two")
	}
	if repeats < 1 {
		repeats = 1
	}
	packedMust(os.MkdirAll(artifactDir, 0700))
	inputSeed := int64(20261019 + d)
	if value := os.Getenv("FHE_PACKED_INPUT_SEED"); value != "" {
		inputSeed, err = strconv.ParseInt(value, 10, 64)
		packedMust(err)
	}
	validation := os.Getenv("FHE_PACKED_VALIDATION") == "1"
	rng := rand.New(rand.NewSource(inputSeed))
	cornerCount := batch / 4
	if d == 8 && batch >= 256+2*d+1 {
		cornerCount = 256
	}
	inputs := make([][]float64, batch)
	reference := make([]float64, batch)
	kinds := map[string]int{}
	maxClearExpansionError := 0.0
	for b := range inputs {
		inputs[b] = make([]float64, d)
		for i := range inputs[b] {
			x := float64(rng.Intn(2049)-1024) / 1024
			if validation {
				switch {
				case b < cornerCount:
					if d == 8 && cornerCount == 256 {
						if (b>>i)&1 == 1 {
							x = 1
						} else {
							x = -1
						}
					} else {
						if rng.Intn(2) == 0 {
							x = -1
						} else {
							x = 1
						}
					}
				case b < cornerCount+2*d:
					coordinate := (b - cornerCount) / 2
					x = 0
					if i == coordinate {
						if (b-cornerCount)%2 == 0 {
							x = 1
						} else {
							x = -1
						}
					}
				case b == cornerCount+2*d:
					x = 0
				}
			} else {
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
			}
			inputs[b][i] = x
		}
		y := m.OutputBias
		for h, row := range m.W {
			z := m.Bias[h]
			for j, w := range row {
				z += w * inputs[b][j]
			}
			y += m.V[h] * z * z
		}
		y *= m.Normalization
		if !networkFinite(y) {
			panic("nonfinite original network reference")
		}
		reference[b] = y
		if len(m.Q) == d && len(m.Linear) == d {
			q := m.Constant
			for i := 0; i < d; i++ {
				if len(m.Q[i]) != d {
					panic("network Q reference dimension mismatch")
				}
				q += m.Linear[i] * inputs[b][i]
				for j := 0; j < d; j++ {
					q += m.Q[i][j] * inputs[b][i] * inputs[b][j]
				}
			}
			if e := math.Abs(q - y); e > maxClearExpansionError {
				maxClearExpansionError = e
			}
		}
		kind := "uniform_dyadic_box"
		if validation {
			switch {
			case b < cornerCount:
				kind = "corners"
			case b < cornerCount+2*d:
				kind = "signed_coordinate"
			case b == cornerCount+2*d:
				kind = "zero"
			}
		} else if b < 5 {
			kind = "fixed_patterns"
		}
		kinds[kind]++
	}
	if maxClearExpansionError > 1e-8 {
		panic(fmt.Sprintf("network clear expansion mismatch %.17g", maxClearExpansionError))
	}
	fixture, err := json.Marshal(map[string]any{"model": m, "inputs": inputs, "reference": reference})
	packedMust(err)
	packedWrite(filepath.Join(artifactDir, "network_fixture.json"), fixture)
	start = time.Now()
	kgen := rlwe.NewKeyGenerator(params)
	sk, pk := kgen.GenKeyPairNew()
	keypairSeconds := packedSecs(start)
	start = time.Now()
	pkBytes, err := pk.MarshalBinary()
	packedMust(err)
	pkSerializeSeconds := packedSecs(start)
	encoder := ckks.NewEncoder(params)
	encryptor := rlwe.NewEncryptor(params, pk)
	decryptor := rlwe.NewDecryptor(params, sk)
	results := []map[string]any{}
	for _, ratio := range []int{-1, 0, 1, 2, 3} {
		name := "network_direct"
		if ratio >= 0 {
			name = fmt.Sprintf("network_bsgs%d", ratio)
		}
		if selected := os.Getenv("FHE_NETWORK_METHOD"); selected != "" && selected != name {
			continue
		}
		runtime.GC()
		heapBefore := packedHeap()
		heapMax := heapBefore
		captureHeap := func() {
			if h := packedHeap(); h > heapMax {
				heapMax = h
			}
		}
		start = time.Now()
		prepared := networkPrepare(params, encoder, m, batch, ratio)
		prepareSeconds := packedSecs(start)
		captureHeap()
		keySet := map[uint64]bool{}
		for _, ltp := range prepared.parameters {
			for _, g := range lintrans.GaloisElements(params, ltp) {
				if g != 1 {
					keySet[g] = true
				}
			}
		}
		reductionRotations := 0
		for step := 1; step < rows; step <<= 1 {
			keySet[params.GaloisElementForRotation(step*batch)] = true
			reductionRotations++
		}
		galels := make([]uint64, 0, len(keySet))
		for g := range keySet {
			galels = append(galels, g)
		}
		sort.Slice(galels, func(i, j int) bool { return galels[i] < galels[j] })
		start = time.Now()
		rlk := kgen.GenRelinearizationKeyNew(sk)
		gks := kgen.GenGaloisKeysNew(galels, sk)
		evk := rlwe.NewMemEvaluationKeySet(rlk, gks...)
		evalKeySeconds := packedSecs(start)
		captureHeap()
		start = time.Now()
		keyBytes, err := evk.MarshalBinary()
		packedMust(err)
		keySerializeSeconds := packedSecs(start)
		captureHeap()
		start = time.Now()
		serverKeys := new(rlwe.MemEvaluationKeySet)
		packedMust(serverKeys.UnmarshalBinary(keyBytes))
		keyDeserializeSeconds := packedSecs(start)
		captureHeap()
		eval := ckks.NewEvaluator(params, serverKeys)
		pt := ckks.NewPlaintext(params, params.MaxLevel())
		pack := func() {
			values := make([]float64, slots)
			for s := range values {
				values[s] = inputs[s%batch][(s/batch)%d]
			}
			packedMust(encoder.Encode(values, pt))
		}
		start = time.Now()
		pack()
		warm, err := encryptor.EncryptNew(pt)
		packedMust(err)
		warmupClientSeconds := packedSecs(start)
		start = time.Now()
		_ = networkServerEvaluate(params, eval, prepared, warm, batch, m.Normalization*m.OutputBias)
		warmupServerSeconds := packedSecs(start)
		samples := map[string][]float64{}
		record := func(stage string, t time.Time) { samples[stage] = append(samples[stage], packedSecs(t)) }
		var inputBytes, outputBytes []byte
		var decodedFirst []float64
		maxAbs, maxImag, sumSquared := 0.0, 0.0, 0.0
		for rep := 0; rep < repeats; rep++ {
			chainStart := time.Now()
			start = time.Now()
			pack()
			record("client_pack_encode_seconds", start)
			start = time.Now()
			clientInput, err := encryptor.EncryptNew(pt)
			packedMust(err)
			record("client_encrypt_seconds", start)
			start = time.Now()
			inputBytes, err = clientInput.MarshalBinary()
			packedMust(err)
			record("input_serialize_seconds", start)
			start = time.Now()
			serverInput := new(rlwe.Ciphertext)
			packedMust(serverInput.UnmarshalBinary(inputBytes))
			record("input_deserialize_seconds", start)
			start = time.Now()
			serverOutput := networkServerEvaluate(params, eval, prepared, serverInput, batch, m.Normalization*m.OutputBias)
			record("server_compute_seconds", start)
			start = time.Now()
			outputBytes, err = serverOutput.MarshalBinary()
			packedMust(err)
			record("output_serialize_seconds", start)
			start = time.Now()
			clientOutput := new(rlwe.Ciphertext)
			packedMust(clientOutput.UnmarshalBinary(outputBytes))
			record("output_deserialize_seconds", start)
			start = time.Now()
			decoded := make([]complex128, slots)
			packedMust(encoder.Decode(decryptor.DecryptNew(clientOutput), decoded))
			record("client_decrypt_decode_seconds", start)
			record("complete_local_chain_seconds", chainStart)
			captureHeap()
			decodedFirst = make([]float64, batch)
			for s, y := range decoded {
				if !networkFinite(real(y)) || !networkFinite(imag(y)) {
					panic(fmt.Sprintf("nonfinite network result %s %s", m.Name, name))
				}
				e := math.Abs(real(y) - reference[s%batch])
				im := math.Abs(imag(y))
				if e > maxAbs {
					maxAbs = e
				}
				if im > maxImag {
					maxImag = im
				}
				sumSquared += e * e
				if s < batch {
					decodedFirst[s] = real(y)
				}
			}
		}
		if maxAbs > 1e-3 || maxImag > 1e-3 {
			panic(fmt.Sprintf("network quality gate failed: %s %s real error %.17g imaginary error %.17g", m.Name, name, maxAbs, maxImag))
		}
		medians := map[string]float64{}
		for stage, values := range samples {
			medians[stage] = packedMedian(values)
		}
		matvecRotations, plainMultiplications := 0, 0
		babyCache := map[int]bool{}
		n1 := []int{}
		for i, ltp := range prepared.parameters {
			plainMultiplications += len(ltp.DiagonalsIndexList)
			n1 = append(n1, prepared.transforms[i].N1)
			if ratio < 0 {
				for _, r := range ltp.DiagonalsIndexList {
					if r != 0 {
						matvecRotations++
					}
				}
			} else {
				_, giants, babies := commonlt.BSGSIndex(ltp.DiagonalsIndexList, slots, prepared.transforms[i].N1)
				for _, r := range giants {
					if r != 0 {
						matvecRotations++
					}
				}
				next := map[int]bool{}
				for _, r := range babies {
					if r != 0 {
						if !babyCache[r] {
							matvecRotations++
						}
						next[r] = true
					}
				}
				babyCache = next
			}
		}
		result := map[string]any{
			"method": name, "representation": "original_signed_square_network", "log_bsgs_ratio": ratio, "bsgs_n1_slots_by_chunk": n1, "hidden_chunks": len(prepared.transforms), "hidden_rows_per_chunk": rows, "hidden_width": hidden, "unused_hidden_rows": len(prepared.transforms)*rows - hidden,
			"quality_passed": true, "required_max_absolute_error": 1e-3, "clear_network_vs_expanded_max_abs_error": maxClearExpansionError,
			"plaintext_multiplications": plainMultiplications, "ciphertext_multiplications": len(prepared.transforms), "relinearizations": 1, "rescalings": len(prepared.transforms) + 1, "variable_variable_depth": 1, "multiplicative_depth": 2, "rescale_levels_consumed": 2, "bootstraps": 0,
			"matvec_rotations": matvecRotations, "reduction_rotations": reductionRotations, "total_rotations": matvecRotations + reductionRotations, "rotation_key_count": len(galels), "galois_elements": galels, "shared_input_decomposition": true, "shared_bsgs_baby_rotations": ratio >= 0,
			"model_prepare_seconds": prepareSeconds, "evaluation_key_generation_seconds": evalKeySeconds, "key_serialize_seconds": keySerializeSeconds, "key_deserialize_seconds": keyDeserializeSeconds, "warmup_server_seconds": warmupServerSeconds, "warmup_client_encode_encrypt_seconds": warmupClientSeconds,
			"evaluation_key_serialized_bytes": len(keyBytes), "encoded_public_model_bytes": prepared.encodedBytes, "encoded_linear_transform_bytes": prepared.transformBytes, "encoded_bias_plaintexts_bytes": prepared.biasBytes, "input_ciphertext_bytes": len(inputBytes), "output_ciphertext_bytes": len(outputBytes), "heap_before_method_bytes": heapBefore, "heap_max_stage_snapshot_bytes": heapMax, "heap_measurement": "stage snapshots of Go HeapAlloc; not process peak RSS",
			"samples": samples, "median": medians, "max_abs_error_all_lanes_repeats": maxAbs, "max_abs_imaginary_all_lanes_repeats": maxImag, "rmse": math.Sqrt(sumSquared / float64(slots*repeats)), "decoded_first_lane": decodedFirst, "key_serialized_sha256": packedHash(keyBytes), "input_sha256_last_repeat": packedHash(inputBytes), "output_sha256_last_repeat": packedHash(outputBytes),
		}
		data, err := json.MarshalIndent(result, "", "  ")
		packedMust(err)
		packedWrite(filepath.Join(artifactDir, name+".json"), data)
		packedWrite(filepath.Join(artifactDir, name+"_input.ct"), inputBytes)
		packedWrite(filepath.Join(artifactDir, name+"_output.ct"), outputBytes)
		results = append(results, result)
		fmt.Printf("network %s %s d=%d h=%d batch=%d server=%.6fs chain=%.6fs error=%.3g keys=%d\n", m.Name, name, d, hidden, batch, medians["server_compute_seconds"], medians["complete_local_chain_seconds"], maxAbs, len(galels))
	}
	if len(results) == 0 {
		panic("unknown FHE_NETWORK_METHOD")
	}
	return map[string]any{"model": m.Name, "dimension": d, "hidden_width": hidden, "batch": batch, "physical_slots": slots, "periodic_input_replications": slots / (d * batch), "hidden_rows_per_chunk": rows, "hidden_chunks": (hidden + rows - 1) / rows, "repeats": repeats, "fixture_sha256": packedHash(fixture), "input_seed": inputSeed, "validation_inputs": validation, "input_kinds": kinds, "parameter_construction_seconds": parameterSeconds, "client_keypair_generation_seconds": keypairSeconds, "public_key_serialize_seconds": pkSerializeSeconds, "public_key_bytes": len(pkBytes), "logN": params.LogN(), "ring_type": params.RingType().String(), "Q": params.Q(), "P": params.P(), "logQP": params.LogQP(), "log_default_scale": params.LogDefaultScale(), "thread_count": 1, "layout": "periodic input slot=j*batch+b; output hidden row chunk*rows+slot/batch; full-row final reduction", "transport": "actual binary encode/decode in one process; network latency unmeasured", "artifact_io": "fixture and evidence disk writes are outside measured inference/setup costs", "one_time_costs": "parameters, public model encode, key generation, key encode/decode; warmup reported separately", "results": results}
}
