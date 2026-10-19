package main

// Strong packed factor comparator: one official diagonal/BSGS linear transform,
// one SIMD square, and one dyadic reduction. Public completion of the square
// absorbs the scalar linear term without an extra ciphertext transform.
import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/tuneinsight/lattigo/v6/circuits/ckks/lintrans"
	commonlt "github.com/tuneinsight/lattigo/v6/circuits/common/lintrans"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type FactorPackedFactors struct {
	Rows  [][]float64 `json:"rows"`
	Signs []int       `json:"signs"`
}

type FactorPackedModel struct {
	Name     string              `json:"name"`
	Q        [][]float64         `json:"Q"`
	Linear   []float64           `json:"linear"`
	Constant float64             `json:"constant"`
	Spectral FactorPackedFactors `json:"spectral"`
	LDL      FactorPackedFactors `json:"ldl"`
}

// Public Gaussian elimination with largest-column pivot, never encrypted data.
func factorPackedShift(m FactorPackedModel) ([]float64, float64, float64) {
	n := len(m.Q)
	augmented := make([][]float64, n)
	for i := range augmented {
		augmented[i] = append(append([]float64(nil), m.Q[i]...), m.Linear[i]/2)
	}
	for k := 0; k < n; k++ {
		pivot := k
		for i := k + 1; i < n; i++ {
			if math.Abs(augmented[i][k]) > math.Abs(augmented[pivot][k]) {
				pivot = i
			}
		}
		if augmented[pivot][k] == 0 || math.IsNaN(augmented[pivot][k]) || math.IsInf(augmented[pivot][k], 0) {
			panic("factor comparator requires nonsingular finite Q")
		}
		augmented[k], augmented[pivot] = augmented[pivot], augmented[k]
		for i := k + 1; i < n; i++ {
			multiplier := augmented[i][k] / augmented[k][k]
			augmented[i][k] = 0
			for j := k + 1; j <= n; j++ {
				augmented[i][j] -= multiplier * augmented[k][j]
			}
		}
	}
	r := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		value := augmented[i][n]
		for j := i + 1; j < n; j++ {
			value -= augmented[i][j] * r[j]
		}
		r[i] = value / augmented[i][i]
		if math.IsNaN(r[i]) || math.IsInf(r[i], 0) {
			panic("nonfinite completion shift")
		}
	}
	quadratic, residual := 0.0, 0.0
	for i := 0; i < n; i++ {
		qr := 0.0
		for j := 0; j < n; j++ {
			qr += m.Q[i][j] * r[j]
		}
		quadratic += r[i] * qr
		residual += math.Abs(2*qr - m.Linear[i])
	}
	return r, m.Constant - quadratic, residual
}

func factorRat(value float64) *big.Rat {
	result := new(big.Rat).SetFloat64(value)
	if result == nil {
		panic("nonfinite factor coefficient")
	}
	return result
}

func factorRatUpper(value *big.Rat) float64 {
	f, _ := value.Float64()
	if factorRat(f).Cmp(value) < 0 {
		f = math.Nextafter(f, math.Inf(1))
	}
	return f
}

// Entire stored affine-square polynomial is expanded in exact rational
// arithmetic on its binary64 coefficients. This includes F, F*r, and c-r^TQr.
func factorPackedResidual(m FactorPackedModel, f FactorPackedFactors, bias []float64, shiftedConstant float64) map[string]any {
	d := len(m.Q)
	n := len(f.Rows)
	rows := make([][]*big.Rat, n)
	biases := make([]*big.Rat, n)
	for k := range rows {
		rows[k] = make([]*big.Rat, d)
		for j := range rows[k] {
			rows[k][j] = factorRat(f.Rows[k][j])
		}
		biases[k] = factorRat(bias[k])
	}
	quadratic := new(big.Rat)
	linear := new(big.Rat)
	for i := 0; i < d; i++ {
		for j := 0; j < d; j++ {
			coefficient := new(big.Rat)
			for k := 0; k < n; k++ {
				term := new(big.Rat).Mul(rows[k][i], rows[k][j])
				if f.Signs[k] < 0 {
					term.Neg(term)
				}
				coefficient.Add(coefficient, term)
			}
			coefficient.Sub(coefficient, factorRat(m.Q[i][j]))
			coefficient.Abs(coefficient)
			quadratic.Add(quadratic, coefficient)
		}
		coefficient := new(big.Rat)
		for k := 0; k < n; k++ {
			term := new(big.Rat).Mul(rows[k][i], biases[k])
			term.Mul(term, big.NewRat(int64(2*f.Signs[k]), 1))
			coefficient.Add(coefficient, term)
		}
		coefficient.Sub(coefficient, factorRat(m.Linear[i]))
		coefficient.Abs(coefficient)
		linear.Add(linear, coefficient)
	}
	constant := factorRat(shiftedConstant)
	for k := 0; k < n; k++ {
		term := new(big.Rat).Mul(biases[k], biases[k])
		if f.Signs[k] < 0 {
			term.Neg(term)
		}
		constant.Add(constant, term)
	}
	constant.Sub(constant, factorRat(m.Constant))
	constant.Abs(constant)
	total := new(big.Rat).Add(quadratic, linear)
	total.Add(total, constant)
	return map[string]any{"quadratic_entry_l1_upper": factorRatUpper(quadratic), "linear_l1_upper": factorRatUpper(linear), "constant_abs_upper": factorRatUpper(constant), "unit_box_absolute_error_upper": factorRatUpper(total), "unit_box_absolute_error_exact": total.RatString(), "arithmetic": "exact math/big.Rat expansion of stored IEEE-754 binary64 affine-square coefficients", "bound": "absolute polynomial error <= R^2*quadratic_entry_l1_upper + R*linear_l1_upper + constant_abs_upper on max(abs(x))<=R", "excludes": "CKKS encoding/encryption/arithmetic/rescaling/decryption error"}
}

// No decryptor/secret key is accepted by this server-side evaluator.
func factorPackedServerEvaluate(params ckks.Parameters, eval *ckks.Evaluator, transform lintrans.LinearTransformation, input *rlwe.Ciphertext, bias *rlwe.Plaintext, constant float64, batch, d int) *rlwe.Ciphertext {
	z, err := lintrans.NewEvaluator(eval).EvaluateNew(input, transform)
	packedMust(err)
	packedMust(eval.Rescale(z, z))
	packedMust(eval.Add(z, bias, z))
	output, err := eval.MulRelinNew(z, z)
	packedMust(err)
	packedMust(eval.Rescale(output, output))
	for step := 1; step < d; step <<= 1 {
		rotated, err := eval.RotateNew(output, step*batch)
		packedMust(err)
		packedMust(eval.Add(output, rotated, output))
	}
	packedMust(eval.Add(output, constant, output))
	return output
}

func RunFactorPacked(m FactorPackedModel, batch, repeats int, artifactDir string) map[string]any {
	oldProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldProcs)
	start := time.Now()
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{LogN: 13, LogQ: []int{58, 40, 40}, LogP: []int{60}, LogDefaultScale: 40})
	packedMust(err)
	parameterSeconds := packedSecs(start)
	d, slots := len(m.Q), params.MaxSlots()
	if d < 2 || d&(d-1) != 0 || slots%d != 0 || len(m.Linear) != d {
		panic("invalid factor model dimensions")
	}
	for i := range m.Q {
		if len(m.Q[i]) != d {
			panic("factor Q is not square")
		}
		for j := range m.Q[i] {
			if math.IsNaN(m.Q[i][j]) || math.IsInf(m.Q[i][j], 0) || math.Abs(m.Q[i][j]-m.Q[j][i]) > 1e-13 {
				panic("invalid factor Q")
			}
		}
	}
	if batch <= 0 {
		batch = slots / d
	}
	if batch*d > slots || slots%(batch*d) != 0 {
		panic("factor batch must divide physical slot layout")
	}
	if repeats < 1 {
		repeats = 1
	}
	packedMust(os.MkdirAll(artifactDir, 0700))
	rng := rand.New(rand.NewSource(20261019 + int64(d)))
	inputs := make([][]float64, batch)
	reference := make([]float64, batch)
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
	packedMust(err)
	packedWrite(filepath.Join(artifactDir, "fixture.json"), fixture)
	inputValues, err := json.Marshal(inputs)
	packedMust(err)
	start = time.Now()
	shift, shiftedConstant, shiftResidual := factorPackedShift(m)
	shiftSeconds := packedSecs(start)
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
		name  string
		kind  string
		ratio int
	}{{"spectral_direct", "spectral", -1}, {"ldl_direct", "ldl", -1}, {"spectral_bsgs0", "spectral", 0}, {"ldl_bsgs0", "ldl", 0}, {"spectral_bsgs1", "spectral", 1}, {"ldl_bsgs1", "ldl", 1}, {"spectral_bsgs2", "spectral", 2}, {"ldl_bsgs2", "ldl", 2}, {"spectral_bsgs3", "spectral", 3}, {"ldl_bsgs3", "ldl", 3}}
	if selected := os.Getenv("FHE_FACTOR_PACKED_METHOD"); selected != "" {
		filtered := methods[:0]
		for _, method := range methods {
			if method.name == selected {
				filtered = append(filtered, method)
			}
		}
		if len(filtered) == 0 {
			panic("unknown FHE_FACTOR_PACKED_METHOD")
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
		f := m.Spectral
		if method.kind == "ldl" {
			f = m.LDL
		}
		if len(f.Rows) > d || len(f.Rows) != len(f.Signs) {
			panic("factor count must fit d rows")
		}
		realBias := make([]float64, len(f.Rows))
		factorMatrix := make([][]complex128, d)
		complexBias := make([]complex128, d)
		for i := 0; i < d; i++ {
			factorMatrix[i] = make([]complex128, d)
			if i >= len(f.Rows) {
				continue
			}
			if len(f.Rows[i]) != d || (f.Signs[i] != 1 && f.Signs[i] != -1) {
				panic("invalid stored factor")
			}
			for j := 0; j < d; j++ {
				realBias[i] += f.Rows[i][j] * shift[j]
			}
			for j := 0; j < d; j++ {
				if f.Signs[i] > 0 {
					factorMatrix[i][j] = complex(f.Rows[i][j], 0)
				} else {
					factorMatrix[i][j] = complex(0, f.Rows[i][j])
				}
			}
			if f.Signs[i] > 0 {
				complexBias[i] = complex(realBias[i], 0)
			} else {
				complexBias[i] = complex(0, realBias[i])
			}
		}
		algebraPrepareSeconds := packedSecs(start)
		start = time.Now()
		residualCertificate := factorPackedResidual(m, f, realBias, shiftedConstant)
		residualCertificateSeconds := packedSecs(start)
		start = time.Now()
		diagonals := make(lintrans.Diagonals[complex128])
		for r := 0; r < d; r++ {
			values := make([]complex128, slots)
			nonzero := false
			for s := range values {
				i := (s / batch) % d
				values[s] = factorMatrix[i][(i+r)%d]
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
		bv := make([]complex128, slots)
		for s := range bv {
			bv[s] = complexBias[(s/batch)%d]
		}
		packedMust(encoder.Encode(bv, affine))
		encodePrepareSeconds := packedSecs(start)
		captureHeap()
		galels := lintrans.GaloisElements(params, ltp)
		seen := map[uint64]bool{}
		for _, g := range galels {
			if g != 1 {
				seen[g] = true
			}
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
		start = time.Now()
		serverKeys := new(rlwe.MemEvaluationKeySet)
		packedMust(serverKeys.UnmarshalBinary(keyBytes))
		keyDeserializeSeconds := packedSecs(start)
		captureHeap()
		eval := ckks.NewEvaluator(params, serverKeys)
		samples := map[string][]float64{}
		record := func(k string, t time.Time) { samples[k] = append(samples[k], packedSecs(t)) }
		maxAbs, maxImag, sumSquared := 0.0, 0.0, 0.0
		allFinite := true
		var inputBytes, outputBytes []byte
		var lastDecoded []float64
		vals := make([]float64, slots)
		for s := range vals {
			vals[s] = inputs[s%batch][(s/batch)%d]
		}
		start = time.Now()
		packedMust(encoder.Encode(vals, pt))
		warmIn, err := encryptor.EncryptNew(pt)
		packedMust(err)
		warmupClientSeconds := packedSecs(start)
		start = time.Now()
		_ = factorPackedServerEvaluate(params, eval, lt, warmIn, affine, shiftedConstant, batch, d)
		warmupSeconds := packedSecs(start)
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
			start = time.Now()
			serverInput := new(rlwe.Ciphertext)
			packedMust(serverInput.UnmarshalBinary(inputBytes))
			record("input_deserialize_seconds", start)
			start = time.Now()
			serverOutput := factorPackedServerEvaluate(params, eval, lt, serverInput, affine, shiftedConstant, batch, d)
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
			lastDecoded = make([]float64, batch)
			for s, y := range decoded {
				if math.IsNaN(real(y)) || math.IsNaN(imag(y)) || math.IsInf(real(y), 0) || math.IsInf(imag(y), 0) {
					allFinite = false
					continue
				}
				e, im := math.Abs(real(y)-reference[s%batch]), math.Abs(imag(y))
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
		gatePassed := allFinite && maxAbs <= 1e-3
		result := map[string]any{
			"method": method.name, "factorization": method.kind, "log_bsgs_ratio": method.ratio, "bsgs_n1_slots": lt.N1,
			"factor_rows": len(f.Rows), "diagonals": len(diagonals), "plaintext_multiplications": len(diagonals), "ciphertext_multiplications": 1, "ciphertext_squares": 1, "relinearizations": 1, "rescalings": 2, "multiplicative_depth": 2, "variable_variable_depth": 1, "rescale_levels_consumed": 2, "bootstraps": 0,
			"matvec_rotations": rotations, "reduction_rotations": reductionRotations, "total_rotations": rotations + reductionRotations, "rotation_key_count": len(galels), "galois_elements": galels,
			"public_algebra_prepare_seconds": algebraPrepareSeconds, "coefficient_certificate_seconds": residualCertificateSeconds, "public_model_encode_seconds": encodePrepareSeconds, "model_prepare_seconds": algebraPrepareSeconds + residualCertificateSeconds + encodePrepareSeconds,
			"evaluation_key_generation_seconds": evalKeySeconds, "key_serialize_seconds": keySerializeSeconds, "key_deserialize_seconds": keyDeserializeSeconds, "warmup_server_seconds": warmupSeconds, "warmup_client_encode_encrypt_seconds": warmupClientSeconds,
			"evaluation_key_serialized_bytes": len(keyBytes), "encoded_linear_transform_bytes": ltBytes, "encoded_affine_bias_bytes": affine.BinarySize(), "input_ciphertext_bytes": len(inputBytes), "output_ciphertext_bytes": len(outputBytes),
			"heap_before_method_bytes": heapBefore, "heap_max_stage_snapshot_bytes": stageHeapMax, "heap_measurement": "stage snapshots of Go HeapAlloc; not process peak RSS",
			"samples": samples, "median": medians, "max_abs_error_all_lanes_repeats": maxAbs, "max_abs_imaginary_all_lanes_repeats": maxImag, "rmse": math.Sqrt(sumSquared / float64(slots*repeats)), "decoded_first_lane": lastDecoded,
			"all_decoded_values_finite": allFinite, "quality_gate_absolute_error": 1e-3, "quality_gate_passed": gatePassed, "client_result_projection": "real part of final decrypted output; imaginary part measured separately",
			"coefficient_residual_certificate": residualCertificate, "completion_shift": shift, "shifted_constant": shiftedConstant, "affine_bias_real": realBias,
			"key_serialized_sha256": packedHash(keyBytes), "input_sha256_last_repeat": packedHash(inputBytes), "output_sha256_last_repeat": packedHash(outputBytes),
		}
		data, err := json.MarshalIndent(result, "", "  ")
		packedMust(err)
		packedWrite(filepath.Join(artifactDir, method.name+".json"), data)
		results = append(results, result)
		fmt.Printf("factorpacked %s %s d=%d batch=%d server=%.6fs chain=%.6fs error=%.3g keys=%d gate=%v\n", m.Name, method.name, d, batch, medians["server_compute_seconds"], medians["complete_local_chain_seconds"], maxAbs, len(galels), gatePassed)
		if !gatePassed {
			panic("factor packed quality gate failed; failure artifact written")
		}
	}
	return map[string]any{"model": m.Name, "dimension": d, "batch": batch, "physical_slots": slots, "periodic_replications": slots / (d * batch), "repeats": repeats, "fixture_sha256": packedHash(fixture), "input_values_sha256": packedHash(inputValues), "parameter_construction_seconds": parameterSeconds, "public_completion_shift_seconds": shiftSeconds, "completion_shift_linear_residual_l1_float": shiftResidual, "client_keypair_generation_seconds": keypairSeconds, "public_key_serialize_seconds": pkSerializeSeconds, "public_key_bytes": len(pkBytes), "logN": params.LogN(), "ring_type": params.RingType().String(), "Q": params.Q(), "P": params.P(), "logQP": params.LogQP(), "log_default_scale": params.LogDefaultScale(), "thread_count": 1, "layout": "slot=i*batch+b, periodic replication if needed", "transport": "actual binary encode/decode in one process; network latency unmeasured", "artifact_io": "fixture and evidence disk writes are outside measured inference/setup costs", "one_time_costs": "parameters, completion shift, public model encode and coefficient certificate, key generation, key encode/decode; warmup reported separately", "results": results}
}
