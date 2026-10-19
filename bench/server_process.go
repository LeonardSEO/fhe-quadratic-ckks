package main

// This server entry point runs after exec in a distinct address space. Its
// stdin schema contains only public parameters/model constants. Key generation,
// secret keys, decryptors and plaintext input fixtures are absent from this API.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/tuneinsight/lattigo/v6/circuits/ckks/lintrans"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type packedServerProcessConfig struct {
	Parameters []byte      `json:"public_parameters"`
	MatrixReal [][]float64 `json:"public_matrix_real"`
	MatrixImag [][]float64 `json:"public_matrix_imag"`
	AffineReal []float64   `json:"public_affine_real"`
	AffineImag []float64   `json:"public_affine_imag"`
	Constant   float64     `json:"public_constant"`
	Batch      int         `json:"batch"`
	Dimension  int         `json:"dimension"`
	BSGSRatio  int         `json:"bsgs_ratio"`
	Square     bool        `json:"square"`
}

type packedServerReady struct {
	URL                  string  `json:"url"`
	PID                  int     `json:"pid"`
	PublicPrepareSeconds float64 `json:"public_prepare_seconds"`
	ReadySeconds         float64 `json:"ready_seconds"`
	PublicConfigSHA256   string  `json:"public_config_sha256"`
	RingType             string  `json:"ring_type"`
	LogQP                float64 `json:"logQP"`
}

type packedServerProcessHandle struct {
	cmd  *exec.Cmd
	info map[string]any
}

func packedRusageBytes(usage *syscall.Rusage) int64 {
	// Darwin reports bytes; Linux reports KiB. This experiment targets Unix.
	if runtime.GOOS == "darwin" {
		return usage.Maxrss
	}
	return usage.Maxrss * 1024
}
func packedServerResourceHeaders(w http.ResponseWriter) {
	w.Header().Set("X-FHE-Server-Pid", strconv.Itoa(os.Getpid()))
	var usage syscall.Rusage
	rss := int64(0)
	if syscall.Getrusage(syscall.RUSAGE_SELF, &usage) == nil {
		rss = packedRusageBytes(&usage)
	}
	w.Header().Set("X-FHE-Server-Peak-Rss-Bytes", strconv.FormatInt(rss, 10))
}

func makePackedServerProcessConfig(params ckks.Parameters, matrix [][]complex128,
	affine []complex128, constant float64, batch, dimension, ratio int, square bool) packedServerProcessConfig {
	encoded, err := params.MarshalBinary()
	packedMust(err)
	c := packedServerProcessConfig{Parameters: encoded, Constant: constant, Batch: batch, Dimension: dimension, BSGSRatio: ratio, Square: square,
		MatrixReal: make([][]float64, dimension), MatrixImag: make([][]float64, dimension), AffineReal: make([]float64, dimension), AffineImag: make([]float64, dimension)}
	for i := 0; i < dimension; i++ {
		c.MatrixReal[i] = make([]float64, dimension)
		c.MatrixImag[i] = make([]float64, dimension)
		for j := 0; j < dimension; j++ {
			c.MatrixReal[i][j] = real(matrix[i][j])
			c.MatrixImag[i][j] = imag(matrix[i][j])
		}
		c.AffineReal[i] = real(affine[i])
		c.AffineImag[i] = imag(affine[i])
	}
	return c
}

// MaybeRunServerProcess MUST be called at the very top of main, before flags,
// client model fixtures, key generation, encryption or decryption are created.
func MaybeRunServerProcess() bool {
	if len(os.Args) != 2 || os.Args[1] != "--fhe-server-process" {
		return false
	}
	runtime.GOMAXPROCS(1)
	readyStart := time.Now()
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 64<<20))
	packedMust(err)
	var config packedServerProcessConfig
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	packedMust(decoder.Decode(&config))
	prepareStart := time.Now()
	var params ckks.Parameters
	packedMust(params.UnmarshalBinary(config.Parameters))
	d, batch, slots := config.Dimension, config.Batch, params.MaxSlots()
	if d < 2 || d&(d-1) != 0 || batch < 1 || d*batch > slots || slots%(d*batch) != 0 {
		panic("invalid public server layout")
	}
	if len(config.MatrixReal) != d || len(config.MatrixImag) != d || len(config.AffineReal) != d || len(config.AffineImag) != d {
		panic("invalid public server matrix")
	}
	diagonals := make(lintrans.Diagonals[complex128])
	for i := 0; i < d; i++ {
		if len(config.MatrixReal[i]) != d || len(config.MatrixImag[i]) != d {
			panic("invalid public server matrix row")
		}
	}
	for r := 0; r < d; r++ {
		values := make([]complex128, slots)
		nonzero := false
		for s := range values {
			i := (s / batch) % d
			j := (i + r) % d
			v := complex(config.MatrixReal[i][j], config.MatrixImag[i][j])
			values[s] = v
			nonzero = nonzero || v != 0
		}
		if nonzero {
			diagonals[r*batch] = values
		}
	}
	pt := ckks.NewPlaintext(params, params.MaxLevel())
	ltp := lintrans.Parameters{DiagonalsIndexList: diagonals.DiagonalsIndexList(), LevelQ: params.MaxLevel(), LevelP: params.MaxLevelP(),
		Scale: rlwe.NewScale(params.Q()[params.MaxLevel()]), LogDimensions: pt.LogDimensions, LogBabyStepGiantStepRatio: config.BSGSRatio}
	transform := lintrans.NewTransformation(params, ltp)
	encoder := ckks.NewEncoder(params)
	packedMust(lintrans.Encode(encoder, diagonals, transform))
	affine := ckks.NewPlaintext(params, params.MaxLevel()-1)
	values := make([]complex128, slots)
	for s := range values {
		i := (s / batch) % d
		values[s] = complex(config.AffineReal[i], config.AffineImag[i])
	}
	packedMust(encoder.Encode(values, affine))
	prepareSeconds := packedSecs(prepareStart)
	server := startPackedPublicHTTPServer(params, transform, affine, config.Constant, batch, d, config.Square)
	defer server.Close()
	ready := packedServerReady{URL: server.URL, PID: os.Getpid(), PublicPrepareSeconds: prepareSeconds, ReadySeconds: packedSecs(readyStart),
		PublicConfigSHA256: packedHash(raw), RingType: params.RingType().String(), LogQP: params.LogQP()}
	packedMust(json.NewEncoder(os.Stdout).Encode(ready))
	// The client explicitly terminates this public server after its benchmark.
	// HTTP remains live while this goroutine blocks; there is no client logic.
	select {}
}

func newPackedSeparateProcessHTTPTransport(config packedServerProcessConfig) *packedHTTPTransport {
	start := time.Now()
	serializeStart := time.Now()
	encoded, err := json.Marshal(config)
	packedMust(err)
	serializeSeconds := packedSecs(serializeStart)
	executable, err := os.Executable()
	packedMust(err)
	command := exec.Command(executable, "--fhe-server-process")
	command.Env = []string{"GOMAXPROCS=1"} // No inherited client credential environment.
	command.Stdin = bytes.NewReader(encoded)
	stdout, err := command.StdoutPipe()
	packedMust(err)
	// Child diagnostics go to the benchmark log; stdout is reserved for one
	// typed readiness record, which cannot contain a secret key by schema.
	command.Stderr = os.Stderr
	packedMust(command.Start())
	type decodedReady struct {
		ready packedServerReady
		err   error
	}
	readyChannel := make(chan decodedReady, 1)
	go func() {
		var ready packedServerReady
		err := json.NewDecoder(stdout).Decode(&ready)
		readyChannel <- decodedReady{ready, err}
	}()
	var ready packedServerReady
	select {
	case result := <-readyChannel:
		if result.err != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
			panic(fmt.Errorf("server child readiness failed (main must call MaybeRunServerProcess first): %w", result.err))
		}
		ready = result.ready
	case <-time.After(30 * time.Second):
		_ = command.Process.Kill()
		_ = command.Wait()
		panic("server child startup exceeded30s")
	}
	if ready.PID == os.Getpid() || ready.PID != command.Process.Pid || ready.PublicConfigSHA256 != packedHash(encoded) {
		_ = command.Process.Kill()
		_ = command.Wait()
		panic("server child identity/config check failed")
	}
	info := map[string]any{"isolation": "separate exec child address space; server receives public model configuration, evaluation keys and ciphertexts only",
		"client_pid": os.Getpid(), "server_pid": ready.PID, "startup_wall_seconds": packedSecs(start), "public_config_serialize_seconds": serializeSeconds,
		"child_public_prepare_seconds": ready.PublicPrepareSeconds, "child_ready_seconds": ready.ReadySeconds,
		"public_config_pipe_bytes": len(encoded), "public_config_sha256": packedHash(encoded), "public_config_transport": "anonymous stdin pipe",
		"startup_cost_relation": "startup_wall_seconds includes config serialize, exec, pipe transfer, child public encoding and listener readiness; do not sum subcomponents again",
		"child_ring_type":       ready.RingType, "child_logQP": ready.LogQP, "memory_measurement": "server PID Getrusage peak RSS; child never executes client key generation/decryption"}
	h := newPackedHTTPClientTransport(ready.URL)
	h.processInfo = info
	h.process = &packedServerProcessHandle{cmd: command, info: info}
	return h
}
func (p *packedServerProcessHandle) Close() {
	_ = p.cmd.Process.Kill()
	_ = p.cmd.Wait()
	if p.cmd.ProcessState != nil {
		if usage, ok := p.cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
			p.info["exit_peak_rss_bytes"] = packedRusageBytes(usage)
		}
	}
}
