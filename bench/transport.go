package main

// Actual HTTP protocol arm. The server constructor and handler receive only
// public parameters/model data and serialized evaluation keys/ciphertexts.
// Neither a secret key nor a decryptor is in this file's API or closure.
import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tuneinsight/lattigo/v6/circuits/ckks/lintrans"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type packedCountConn struct {
	net.Conn
	sent, received *atomic.Int64
}

func (c *packedCountConn) Write(p []byte) (int, error) {
	n, e := c.Conn.Write(p)
	c.sent.Add(int64(n))
	return n, e
}
func (c *packedCountConn) Read(p []byte) (int, error) {
	n, e := c.Conn.Read(p)
	c.received.Add(int64(n))
	return n, e
}

type packedHTTPTransport struct {
	server         *httptest.Server
	client         *http.Client
	transport      *http.Transport
	baseURL        string
	process        *packedServerProcessHandle
	processInfo    map[string]any
	closeOnce      sync.Once
	sent, received atomic.Int64
}

func packedHeader(w http.ResponseWriter, name string, seconds float64) {
	w.Header().Set(name, strconv.FormatFloat(seconds, 'g', 17, 64))
}
func packedResponseMetric(r *http.Response, name string) float64 {
	v, e := strconv.ParseFloat(r.Header.Get(name), 64)
	packedMust(e)
	return v
}

func startPackedPublicHTTPServer(params ckks.Parameters, transform lintrans.LinearTransformation,
	linear *rlwe.Plaintext, constant float64, batch, dimension int, square bool) *httptest.Server {
	var evaluator *ckks.Evaluator
	var evaluatorMutex sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", 405)
			return
		}
		start := time.Now()
		data, err := io.ReadAll(io.LimitReader(r.Body, 256<<20))
		packedMust(err)
		readSeconds := packedSecs(start)
		start = time.Now()
		evaluatorMutex.Lock()
		defer evaluatorMutex.Unlock()
		keys := new(rlwe.MemEvaluationKeySet)
		packedMust(keys.UnmarshalBinary(data))
		evaluator = ckks.NewEvaluator(params, keys)
		decodeSeconds := packedSecs(start)
		packedHeader(w, "X-FHE-Key-Deserialize-Seconds", decodeSeconds)
		packedHeader(w, "X-FHE-Body-Read-Seconds", readSeconds)
		packedServerResourceHeaders(w)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "2")
		_, err = w.Write([]byte("OK"))
		packedMust(err)
	})
	mux.HandleFunc("/infer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", 405)
			return
		}
		evaluatorMutex.Lock()
		defer evaluatorMutex.Unlock()
		if evaluator == nil {
			http.Error(w, "evaluation keys required", 409)
			return
		}
		start := time.Now()
		data, err := io.ReadAll(io.LimitReader(r.Body, 16<<20))
		packedMust(err)
		readSeconds := packedSecs(start)
		start = time.Now()
		input := new(rlwe.Ciphertext)
		packedMust(input.UnmarshalBinary(data))
		decodeSeconds := packedSecs(start)
		start = time.Now()
		output := packedServerEvaluate(params, evaluator, transform, input, linear, constant, batch, dimension, square)
		computeSeconds := packedSecs(start)
		start = time.Now()
		result, err := output.MarshalBinary()
		packedMust(err)
		encodeSeconds := packedSecs(start)
		packedHeader(w, "X-FHE-Input-Deserialize-Seconds", decodeSeconds)
		packedHeader(w, "X-FHE-Server-Compute-Seconds", computeSeconds)
		packedHeader(w, "X-FHE-Output-Serialize-Seconds", encodeSeconds)
		packedHeader(w, "X-FHE-Body-Read-Seconds", readSeconds)
		packedServerResourceHeaders(w)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(result)))
		_, err = w.Write(result)
		packedMust(err)
	})
	return httptest.NewServer(mux)
}

func newPackedHTTPClientTransport(baseURL string) *packedHTTPTransport {
	h := &packedHTTPTransport{baseURL: baseURL, processInfo: map[string]any{"isolation": "same process; memory measurements combine client and server"}}
	dialer := new(net.Dialer)
	h.transport = &http.Transport{DisableCompression: true, MaxIdleConns: 1, MaxIdleConnsPerHost: 1,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			c, e := dialer.DialContext(ctx, network, address)
			if e != nil {
				return nil, e
			}
			return &packedCountConn{Conn: c, sent: &h.sent, received: &h.received}, nil
		},
	}
	h.client = &http.Client{Transport: h.transport, Timeout: 2 * time.Minute}
	return h
}

func newPackedHTTPTransport(params ckks.Parameters, transform lintrans.LinearTransformation,
	linear *rlwe.Plaintext, constant float64, batch, dimension int, square bool) *packedHTTPTransport {
	server := startPackedPublicHTTPServer(params, transform, linear, constant, batch, dimension, square)
	h := newPackedHTTPClientTransport(server.URL)
	h.server = server
	return h
}

func (h *packedHTTPTransport) DistributeKeys(keys []byte) map[string]any {
	sent, received := h.sent.Load(), h.received.Load()
	start := time.Now()
	response, err := h.client.Post(h.baseURL+"/keys", "application/octet-stream", bytes.NewReader(keys))
	packedMust(err)
	body, err := io.ReadAll(response.Body)
	packedMust(err)
	packedMust(response.Body.Close())
	if response.StatusCode != 200 || string(body) != "OK" {
		panic(fmt.Sprintf("key HTTP status %d", response.StatusCode))
	}
	return map[string]any{"roundtrip_seconds": packedSecs(start), "server_deserialize_seconds": packedResponseMetric(response, "X-FHE-Key-Deserialize-Seconds"), "server_read_body_seconds": packedResponseMetric(response, "X-FHE-Body-Read-Seconds"), "client_sent_http_bytes": h.sent.Load() - sent, "client_received_http_bytes": h.received.Load() - received, "serialized_key_payload_bytes": len(keys), "server_process_id": packedResponseMetric(response, "X-FHE-Server-Pid"), "server_peak_rss_bytes": packedResponseMetric(response, "X-FHE-Server-Peak-Rss-Bytes")}
}

func (h *packedHTTPTransport) Infer(input []byte) ([]byte, map[string]float64) {
	sent, received := h.sent.Load(), h.received.Load()
	start := time.Now()
	response, err := h.client.Post(h.baseURL+"/infer", "application/octet-stream", bytes.NewReader(input))
	packedMust(err)
	result, err := io.ReadAll(response.Body)
	packedMust(err)
	packedMust(response.Body.Close())
	if response.StatusCode != 200 {
		panic(fmt.Sprintf("inference HTTP status %d", response.StatusCode))
	}
	return result, map[string]float64{
		"http_roundtrip_seconds":        packedSecs(start),
		"input_deserialize_seconds":     packedResponseMetric(response, "X-FHE-Input-Deserialize-Seconds"),
		"server_compute_seconds":        packedResponseMetric(response, "X-FHE-Server-Compute-Seconds"),
		"output_serialize_seconds":      packedResponseMetric(response, "X-FHE-Output-Serialize-Seconds"),
		"server_http_read_body_seconds": packedResponseMetric(response, "X-FHE-Body-Read-Seconds"),
		"server_process_id":             packedResponseMetric(response, "X-FHE-Server-Pid"), "server_peak_rss_bytes": packedResponseMetric(response, "X-FHE-Server-Peak-Rss-Bytes"),
		"client_sent_http_bytes": float64(h.sent.Load() - sent), "client_received_http_bytes": float64(h.received.Load() - received),
	}
}
func (h *packedHTTPTransport) Close() {
	h.closeOnce.Do(func() {
		start := time.Now()
		h.transport.CloseIdleConnections()
		if h.server != nil {
			h.server.Close()
		}
		if h.process != nil {
			h.process.Close()
		}
		h.processInfo["shutdown_seconds"] = packedSecs(start)
	})
}
