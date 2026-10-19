package main

// Packed scalar-quadratic evaluation. The server evaluator below receives no
// secret key. Client/server boundaries are exercised by binary serialization;
// this in-process benchmark measures bytes and codec cost, not network latency.
import (
	"crypto/sha256"
	"encoding/hex"
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

type PackedFactorization struct {
	Rows  [][]float64 `json:"rows"`
	Signs []int       `json:"signs"`
}

type PackedModel struct {
	Name          string               `json:"name"`
	Q             [][]float64          `json:"Q"`
	Linear        []float64            `json:"linear"`
	Constant      float64              `json:"constant"`
	Spectral      *PackedFactorization `json:"spectral,omitempty"`
	LDL           *PackedFactorization `json:"ldl,omitempty"`
	AffineShift   []float64            `json:"affine_shift,omitempty"`
	ShiftConstant float64              `json:"shift_constant,omitempty"`
}

func packedMust(err error) {
	if err != nil {
		panic(err)
	}
}
func packedSecs(start time.Time) float64 { return time.Since(start).Seconds() }
func packedMedian(v []float64) float64 {
	w := append([]float64(nil), v...)
	sort.Float64s(w)
	if len(w) == 0 {
		return 0
	}
	if len(w)%2 == 1 {
		return w[len(w)/2]
	}
	return (w[len(w)/2-1] + w[len(w)/2]) / 2
}
func packedHash(v []byte) string           { s := sha256.Sum256(v); return hex.EncodeToString(s[:]) }
func packedWrite(path string, data []byte) { packedMust(os.WriteFile(path, data, 0600)) }
func packedHeap() uint64                   { var s runtime.MemStats; runtime.ReadMemStats(&s); return s.HeapAlloc }

// packedServerEvaluate has no decryptor or secret-key argument. All operations
// between input and returned final ciphertext are homomorphic.
func packedServerEvaluate(params ckks.Parameters, eval *ckks.Evaluator,
	transform lintrans.LinearTransformation, input *rlwe.Ciphertext,
	linear *rlwe.Plaintext, constant float64, batch, dimension int, square bool) *rlwe.Ciphertext {
	z, err := lintrans.NewEvaluator(eval).EvaluateNew(input, transform)
	packedMust(err)
	packedMust(eval.Rescale(z, z))
	packedMust(eval.Add(z, linear, z))
	x := z
	if !square {
		x = eval.DropLevelNew(input, input.Level()-z.Level())
	}
	result, err := eval.MulRelinNew(x, z)
	packedMust(err)
	packedMust(eval.Rescale(result, result))
	for step := 1; step < dimension; step <<= 1 {
		r, err := eval.RotateNew(result, step*batch)
		packedMust(err)
		packedMust(eval.Add(result, r, result))
	}
	packedMust(eval.Add(result, constant, result))
	return result
}

// RunPacked compares the exact same scalar polynomial, slot layout, inputs,
// cryptographic parameters and full client/server chain. All fifteen predetermined
// direct/official-BSGS settings (including triangular symmetry folding) are retained so baseline tuning is visible.
func RunPacked(m PackedModel, batch, repeats int, artifactDir string) map[string]any {
	return RunPackedWithRing(m, batch, repeats, artifactDir, false)
}

// RunPackedWithRing holds LogN, modulus sizes, application batch and quality
// requirements fixed when selecting the conjugate-invariant real ring. A zero
// batch means 4096/d in either ring: CI's additional slots repeat the same data.
func RunPackedWithRing(m PackedModel, batch, repeats int, artifactDir string, ci bool) map[string]any {
	oldProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldProcs)
	start := time.Now()
	ringType := ring.Standard
	if ci {
		ringType = ring.ConjugateInvariant
	}
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN: 13, LogQ: []int{58, 40, 40}, LogP: []int{60}, LogDefaultScale: 40, RingType: ringType,
	})
	packedMust(err)
	parameterSeconds := packedSecs(start)
	transportMode := os.Getenv("FHE_PACKED_TRANSPORT")
	separateServer := os.Getenv("FHE_PACKED_SERVER_PROCESS") == "1"
	if separateServer {
		transportMode = "http"
	}
	if transportMode != "" && transportMode != "codec" && transportMode != "http" {
		panic("FHE_PACKED_TRANSPORT must be codec or http")
	}
	transportLabel := "actual binary encode/decode in one process; network latency unmeasured"
	if transportMode == "http" {
		transportLabel = "actual same-process loopback HTTP/TCP; shared client/server memory; key POST+installation, ciphertext POST+response; keepalive; HTTP bytes exclude TCP/IP headers"
		if separateServer {
			transportLabel = "actual separate-exec-child loopback HTTP/TCP; server address space receives no client secret key; public config stdin pipe and key POST/install; ciphertext POST/response; HTTP bytes exclude TCP/IP headers"
		}
	}
	d := len(m.Q)
	slots := params.MaxSlots()
	if d < 2 || d&(d-1) != 0 || slots%d != 0 {
		panic("packed dimension must be a power of two >=2 dividing slots")
	}
	if len(m.Linear) != d {
		panic("packed linear dimension mismatch")
	}
	for i := 0; i < d; i++ {
		if len(m.Q[i]) != d {
			panic("packed Q is not square")
		}
		for j := 0; j < d; j++ {
			if math.IsNaN(m.Q[i][j]) || math.IsInf(m.Q[i][j], 0) {
				panic("nonfinite packed Q")
			}
			if math.Abs(m.Q[i][j]-m.Q[j][i]) > 1e-13 {
				panic("packed Q must be symmetric")
			}
		}
	}
	if batch <= 0 {
		batch = (1 << (params.LogN() - 1)) / d
	}
	if batch*d > slots || slots%(batch*d) != 0 {
		panic("packed batch must permit exact periodic replication in slots")
	}
	if repeats < 1 {
		repeats = 1
	}
	packedMust(os.MkdirAll(artifactDir, 0700))
	// Inputs are dyadic, deterministic benchmark fixtures, never cryptographic RNG.
	inputSeed := int64(20261019 + d)
	if v := os.Getenv("FHE_PACKED_INPUT_SEED"); v != "" {
		inputSeed, err = strconv.ParseInt(v, 10, 64)
		packedMust(err)
	}
	validation := os.Getenv("FHE_PACKED_VALIDATION") == "1"
	rng := rand.New(rand.NewSource(inputSeed))
	inputKinds := map[string]int{}
	cornerCount := batch / 4
	if d == 8 && batch >= 256+2*d+1 {
		cornerCount = 256
	}
	inputs := make([][]float64, batch)
	reference := make([]float64, batch)
	for b := 0; b < batch; b++ {
		inputs[b] = make([]float64, d)
		for i := 0; i < d; i++ {
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
		y := m.Constant
		for i := 0; i < d; i++ {
			y += m.Linear[i] * inputs[b][i]
			for j := 0; j < d; j++ {
				y += m.Q[i][j] * inputs[b][i] * inputs[b][j]
			}
		}
		reference[b] = y
		if math.IsNaN(y) || math.IsInf(y, 0) {
			panic("nonfinite clear reference")
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
		inputKinds[kind]++
	}
	fixture, _ := json.Marshal(map[string]any{"model": m, "inputs": inputs, "reference": reference})
	packedWrite(filepath.Join(artifactDir, "fixture.json"), fixture)
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
	methods := []struct {
		form  string
		ratio int
		name  string
	}{
		{"symmetric", -1, "symmetric_direct"}, {"oriented", -1, "oriented_direct"}, {"triangular", -1, "triangular_direct"},
		{"symmetric", 0, "symmetric_bsgs0"}, {"oriented", 0, "oriented_bsgs0"}, {"triangular", 0, "triangular_bsgs0"},
		{"symmetric", 1, "symmetric_bsgs1"}, {"oriented", 1, "oriented_bsgs1"}, {"triangular", 1, "triangular_bsgs1"},
		{"symmetric", 2, "symmetric_bsgs2"}, {"oriented", 2, "oriented_bsgs2"}, {"triangular", 2, "triangular_bsgs2"},
		{"symmetric", 3, "symmetric_bsgs3"}, {"oriented", 3, "oriented_bsgs3"}, {"triangular", 3, "triangular_bsgs3"},
	}
	if len(m.AffineShift) == d {
		for _, candidate := range []struct {
			name string
			f    *PackedFactorization
		}{{"spectral", m.Spectral}, {"ldl", m.LDL}} {
			if candidate.f == nil || len(candidate.f.Rows) > d || len(candidate.f.Signs) != len(candidate.f.Rows) {
				continue
			}
			supported := true
			for _, sign := range candidate.f.Signs {
				if ci && sign < 0 {
					supported = false
				}
			}
			if supported {
				for _, ratio := range []int{-1, 0, 1, 2, 3} {
					suffix := "direct"
					if ratio >= 0 {
						suffix = fmt.Sprintf("bsgs%d", ratio)
					}
					methods = append(methods, struct {
						form  string
						ratio int
						name  string
					}{candidate.name, ratio, candidate.name + "_affine_square_" + suffix})
				}
			}
		}
	}
	if selected := os.Getenv("FHE_PACKED_METHOD"); selected != "" {
		filtered := methods[:0]
		for _, method := range methods {
			if method.name == selected {
				filtered = append(filtered, method)
			}
		}
		if len(filtered) == 0 {
			panic("unknown FHE_PACKED_METHOD")
		}
		methods = filtered
	}
	results := make([]map[string]any, 0, len(methods))
	for _, method := range methods {
		runtime.GC()
		heapBefore := packedHeap()
		stageHeapMax := heapBefore
		captureHeap := func() {
			h := packedHeap()
			if h > stageHeapMax {
				stageHeapMax = h
			}
		}
		start = time.Now()
		square := method.form == "spectral" || method.form == "ldl"
		A := make([][]complex128, d)
		affineValues := make([]complex128, d)
		for i := range A {
			A[i] = make([]complex128, d)
		}
		constant := m.Constant
		if square {
			factors := m.Spectral
			if method.form == "ldl" {
				factors = m.LDL
			}
			constant = m.ShiftConstant
			for i, row := range factors.Rows {
				if len(row) != d {
					panic("factor row dimension mismatch")
				}
				sign := factors.Signs[i]
				coefficient := cmplx.Sqrt(complex(float64(sign), 0))
				for j, value := range row {
					A[i][j] = coefficient * complex(value, 0)
					affineValues[i] += A[i][j] * complex(m.AffineShift[j], 0)
				}
			}
		} else {
			for i := 0; i < d; i++ {
				affineValues[i] = complex(m.Linear[i], 0)
				for j := 0; j < d; j++ {
					q := m.Q[i][j]
					switch method.form {
					case "symmetric":
						A[i][j] = complex(q, 0)
					case "triangular":
						if i == j {
							A[i][j] = complex(q, 0)
						} else if i < j {
							A[i][j] = complex(2*q, 0)
						}
					case "oriented":
						r := (j - i + d) % d
						switch {
						case r == 0:
							A[i][j] = complex(q, 0)
						case r < d/2:
							A[i][j] = complex(2*q, 0)
						case r == d/2:
							A[i][j] = complex(q, 0)
						}
					default:
						panic("unknown packed representation")
					}
				}
			}
		}
		residual, affineResidual, constantResidual := 0.0, 0.0, 0.0
		for i := 0; i < d; i++ {
			for j := 0; j < d; j++ {
				coefficient := (A[i][j] + A[j][i]) / 2
				if square {
					coefficient = 0
					for k := 0; k < d; k++ {
						coefficient += A[k][i] * A[k][j]
					}
				}
				residual += math.Abs(m.Q[i][j] - real(coefficient))
			}
		}
		if square {
			reconstructedConstant := complex(constant, 0)
			for k := 0; k < d; k++ {
				reconstructedConstant += affineValues[k] * affineValues[k]
			}
			constantResidual = math.Abs(m.Constant - real(reconstructedConstant))
			for j := 0; j < d; j++ {
				coefficient := complex(0, 0)
				for k := 0; k < d; k++ {
					coefficient += 2 * A[k][j] * affineValues[k]
				}
				affineResidual += math.Abs(m.Linear[j] - real(coefficient))
			}
		}
		diagonals := make(lintrans.Diagonals[complex128])
		for r := 0; r < d; r++ {
			values := make([]complex128, slots)
			nonzero := false
			for s := range values {
				i := (s / batch) % d
				values[s] = A[i][(i+r)%d]
				nonzero = nonzero || values[s] != 0
			}
			if nonzero {
				diagonals[r*batch] = values
			}
		}
		pt := ckks.NewPlaintext(params, params.MaxLevel())
		ltp := lintrans.Parameters{DiagonalsIndexList: diagonals.DiagonalsIndexList(), LevelQ: params.MaxLevel(), LevelP: params.MaxLevelP(), Scale: rlwe.NewScale(params.Q()[params.MaxLevel()]), LogDimensions: pt.LogDimensions, LogBabyStepGiantStepRatio: method.ratio}
		lt := lintrans.NewTransformation(params, ltp)
		packedMust(lintrans.Encode(encoder, diagonals, lt))
		affine := ckks.NewPlaintext(params, params.MaxLevel()-1)
		lv := make([]complex128, slots)
		for s := range lv {
			lv[s] = affineValues[(s/batch)%d]
		}
		packedMust(encoder.Encode(lv, affine))
		prepareSeconds := packedSecs(start)
		captureHeap()
		galels := lintrans.GaloisElements(params, ltp)
		seen := map[uint64]bool{}
		for _, g := range galels {
			if g != 1 {
				seen[g] = true
			} // identity needs no evaluation key
		}
		for step := 1; step < d; step <<= 1 {
			seen[params.GaloisElementForRotation(step*batch)] = true
		}
		galels = galels[:0]
		for g := range seen {
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
		var eval *ckks.Evaluator
		var httpTransport *packedHTTPTransport
		var httpKeyDistribution map[string]any
		keyDeserializeSeconds, httpSetupSeconds := 0.0, 0.0
		if transportMode == "http" {
			start = time.Now()
			if separateServer {
				config := makePackedServerProcessConfig(params, A, affineValues, constant, batch, d, method.ratio, square)
				httpTransport = newPackedSeparateProcessHTTPTransport(config)
			} else {
				httpTransport = newPackedHTTPTransport(params, lt, affine, constant, batch, d, square)
			}
			defer httpTransport.Close()
			httpSetupSeconds = packedSecs(start)
			httpKeyDistribution = httpTransport.DistributeKeys(keyBytes)
			keyDeserializeSeconds = httpKeyDistribution["server_deserialize_seconds"].(float64)
		} else {
			start = time.Now()
			serverKeys := new(rlwe.MemEvaluationKeySet)
			packedMust(serverKeys.UnmarshalBinary(keyBytes))
			keyDeserializeSeconds = packedSecs(start)
			eval = ckks.NewEvaluator(params, serverKeys)
		}
		captureHeap()
		samples := map[string][]float64{}
		record := func(k string, s time.Time) { samples[k] = append(samples[k], packedSecs(s)) }
		maxAbs, maxImag, sumSquared := 0.0, 0.0, 0.0
		var inputBytes, outputBytes []byte
		var lastDecoded []float64
		// One explicit warmup is charged separately from timed inference.
		vals := make([]float64, slots)
		for s := range vals {
			vals[s] = inputs[s%batch][(s/batch)%d]
		}
		start = time.Now()
		packedMust(encoder.Encode(vals, pt))
		warmIn, err := encryptor.EncryptNew(pt)
		packedMust(err)
		warmupClientSeconds := packedSecs(start)
		var warmupHTTP map[string]float64
		start = time.Now()
		if httpTransport != nil {
			wire, err := warmIn.MarshalBinary()
			packedMust(err)
			_, warmupHTTP = httpTransport.Infer(wire)
		} else {
			_ = packedServerEvaluate(params, eval, lt, warmIn, affine, constant, batch, d, square)
		}
		warmupSeconds := packedSecs(start)
		warmupServerSeconds := warmupSeconds
		if warmupHTTP != nil {
			warmupServerSeconds = warmupHTTP["server_compute_seconds"]
		}
		for rep := 0; rep < repeats; rep++ {
			chainStart := time.Now()
			start = time.Now()
			vals = make([]float64, slots)
			for s := range vals {
				vals[s] = inputs[s%batch][(s/batch)%d]
			}
			packedMust(encoder.Encode(vals, pt))
			record("client_pack_encode_seconds", start)
			start = time.Now()
			clientInput, err := encryptor.EncryptNew(pt)
			packedMust(err)
			record("client_encrypt_seconds", start)
			start = time.Now()
			inputBytes, err = clientInput.MarshalBinary()
			packedMust(err)
			record("input_serialize_seconds", start)
			if httpTransport != nil {
				var metrics map[string]float64
				outputBytes, metrics = httpTransport.Infer(inputBytes)
				for key, value := range metrics {
					samples[key] = append(samples[key], value)
				}
			} else {
				start = time.Now()
				serverInput := new(rlwe.Ciphertext)
				packedMust(serverInput.UnmarshalBinary(inputBytes))
				record("input_deserialize_seconds", start)
				start = time.Now()
				serverOutput := packedServerEvaluate(params, eval, lt, serverInput, affine, constant, batch, d, square)
				record("server_compute_seconds", start)
				start = time.Now()
				outputBytes, err = serverOutput.MarshalBinary()
				packedMust(err)
				record("output_serialize_seconds", start)
			}
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
			lastDecoded = make([]float64, batch)
			// Check all replicated output lanes, not only the first lane.
			for s, y := range decoded {
				b := s % batch
				if math.IsNaN(real(y)) || math.IsInf(real(y), 0) || math.IsNaN(imag(y)) || math.IsInf(imag(y), 0) {
					panic(fmt.Sprintf("nonfinite encrypted result %s %s", m.Name, method.name))
				}
				e := math.Abs(real(y) - reference[b])
				im := math.Abs(imag(y))
				if e > maxAbs {
					maxAbs = e
				}
				if im > maxImag {
					maxImag = im
				}
				sumSquared += e * e
				if s < batch {
					lastDecoded[s] = real(y)
				}
			}
			if rep == repeats-1 {
				packedWrite(filepath.Join(artifactDir, method.name+"_input.ct"), inputBytes)
				packedWrite(filepath.Join(artifactDir, method.name+"_output.ct"), outputBytes)
			}
		}
		if maxAbs > 1e-3 {
			panic(fmt.Sprintf("quality gate failed: %s %s max error %.17g >1e-3", m.Name, method.name, maxAbs))
		}
		medians := map[string]float64{}
		for k, v := range samples {
			medians[k] = packedMedian(v)
		}
		rotations := len(diagonals) - 1
		if method.ratio >= 0 {
			_, giants, babies := commonlt.BSGSIndex(ltp.DiagonalsIndexList, slots, lt.N1)
			rotations = 0
			for _, r := range giants {
				if r != 0 {
					rotations++
				}
			}
			for _, r := range babies {
				if r != 0 {
					rotations++
				}
			}
		}
		reductionRotations := 0
		for step := 1; step < d; step <<= 1 {
			reductionRotations++
		}
		ltBytes := 0
		for _, p := range lt.Vec {
			ltBytes += p.BinarySize()
		}
		if httpTransport != nil {
			httpTransport.Close()
		}
		var serverProcessInfo map[string]any
		if httpTransport != nil {
			serverProcessInfo = httpTransport.processInfo
		}
		result := map[string]any{
			"server_process": serverProcessInfo,
			"method":         method.name, "oriented": method.form == "oriented", "representation": method.form, "square_kernel": square, "log_bsgs_ratio": method.ratio, "bsgs_n1_slots": lt.N1,
			"coefficient_residual_entry_l1": residual, "affine_residual_entry_l1_float": affineResidual, "constant_residual_float": constantResidual, "quality_passed": true, "required_max_absolute_error": 1e-3, "diagonals": len(diagonals), "plaintext_multiplications": len(diagonals), "ciphertext_multiplications": 1, "relinearizations": 1, "rescalings": 2, "multiplicative_depth": 2, "variable_variable_depth": 1, "rescale_levels_consumed": 2, "bootstraps": 0,
			"matvec_rotations": rotations, "reduction_rotations": reductionRotations, "total_rotations": rotations + reductionRotations, "rotation_key_count": len(galels), "galois_elements": galels,
			"transport": transportLabel, "http_setup_seconds": httpSetupSeconds, "http_key_distribution": httpKeyDistribution, "warmup_http": warmupHTTP,
			"model_prepare_seconds": prepareSeconds, "evaluation_key_generation_seconds": evalKeySeconds, "key_serialize_seconds": keySerializeSeconds, "key_deserialize_seconds": keyDeserializeSeconds, "warmup_server_seconds": warmupServerSeconds, "warmup_total_seconds": warmupSeconds, "warmup_client_encode_encrypt_seconds": warmupClientSeconds,
			"evaluation_key_serialized_bytes": len(keyBytes), "encoded_linear_transform_bytes": ltBytes, "input_ciphertext_bytes": len(inputBytes), "output_ciphertext_bytes": len(outputBytes),
			"heap_before_method_bytes": heapBefore, "heap_max_stage_snapshot_bytes": stageHeapMax, "heap_measurement": "stage snapshots of Go HeapAlloc; not process peak RSS",
			"samples": samples, "median": medians, "max_abs_error_all_lanes_repeats": maxAbs, "max_abs_imaginary_all_lanes_repeats": maxImag, "rmse": math.Sqrt(sumSquared / float64(slots*repeats)), "decoded_first_lane": lastDecoded,
			"key_serialized_sha256": packedHash(keyBytes), "input_sha256_last_repeat": packedHash(inputBytes), "output_sha256_last_repeat": packedHash(outputBytes),
		}
		data, err := json.MarshalIndent(result, "", "  ")
		packedMust(err)
		packedWrite(filepath.Join(artifactDir, method.name+".json"), data)
		results = append(results, result)
		if httpTransport != nil {
			httpTransport.Close()
		}
		fmt.Printf("packed %s %s d=%d batch=%d server=%.6fs chain=%.6fs error=%.3g keys=%d\n", m.Name, method.name, d, batch, medians["server_compute_seconds"], medians["complete_local_chain_seconds"], maxAbs, len(galels))
	}
	return map[string]any{"model": m.Name, "dimension": d, "batch": batch, "physical_slots": slots, "periodic_replications": slots / (d * batch), "repeats": repeats, "fixture_sha256": packedHash(fixture), "input_seed": inputSeed, "validation_inputs": validation, "input_kinds": inputKinds, "parameter_construction_seconds": parameterSeconds, "client_keypair_generation_seconds": keypairSeconds, "public_key_serialize_seconds": pkSerializeSeconds, "public_key_bytes": len(pkBytes), "logN": params.LogN(), "ring_type": params.RingType().String(), "Q": params.Q(), "P": params.P(), "logQP": params.LogQP(), "log_default_scale": params.LogDefaultScale(), "thread_count": 1, "server_process_requested": separateServer, "layout": "slot=i*batch+b, periodic replication if needed", "transport": transportLabel, "artifact_io": "fixture and evidence disk writes are outside measured inference/setup costs", "one_time_costs": "parameters, public model encode, key generation, key serialize; codec key deserialize OR HTTP setup+key roundtrip (HTTP roundtrip already includes key deserialize); warmup reported separately", "results": results}
}
