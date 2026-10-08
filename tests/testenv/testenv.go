package testenv

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

// Env descreve a API viva sob teste.
type Env struct {
	BaseURL     string
	APIKey      string
	ScrapersKey string
}

// Start sobe a stack e agenda a derrubada via t.Cleanup.
func Start(t *testing.T) *Env {
	t.Helper()
	env, cleanup, err := StartContext(context.Background())
	if err != nil {
		t.Fatalf("testenv: %v\n%s", err, logTail())
	}
	t.Cleanup(cleanup)
	return env
}

var lastLogs []byte

func logTail() string {
	if len(lastLogs) > 2000 {
		return string(lastLogs[len(lastLogs)-2000:])
	}
	return string(lastLogs)
}

// StartContext sobe Mongo + API e devolve o ambiente e a função de limpeza.
// O binário herda apenas as variáveis listadas abaixo: credenciais da shell
// (ex.: .env de produção) nunca vazam para o processo sob teste.
func StartContext(ctx context.Context) (*Env, func(), error) {
	mongoContainer, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		return nil, nil, fmt.Errorf("mongo container: %w", err)
	}
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = mongoContainer.Terminate(ctx)
	}

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("mongo uri: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("free port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	workdir, err := os.MkdirTemp("", "circuito-api-test-*")
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("workdir: %w", err)
	}
	cleanupAll := func() {
		os.RemoveAll(workdir)
		cleanup()
	}

	binary := filepath.Join(workdir, "circuito-api")
	build := exec.Command("go", "build", "-o", binary, "github.com/marcosarl1/Circuito-API/cmd/api")
	build.Dir = moduleRoot()
	if output, err := build.CombinedOutput(); err != nil {
		cleanupAll()
		return nil, nil, fmt.Errorf("go build: %w\n%s", err, output)
	}

	const apiKey = "test-api-key"
	const scrapersKey = "test-scrapers-key"

	command := exec.Command(binary)
	command.Dir = workdir // sem .env aqui: config vem só das vars abaixo
	command.Env = []string{
		"MONGODB_URI=" + mongoURI,
		"MONGODB_DB_NAME=integrationtest",
		"MONGODB_COLLECTION=eventos",
		"API_PORT=" + strconv.Itoa(port),
		"API_KEY=" + apiKey,
		"SCRAPERS_API_KEY=" + scrapersKey,
		"CORS_ORIGINS=*",
	}
	logWriter := &ringBuffer{limit: 64 * 1024}
	command.Stdout = logWriter
	command.Stderr = logWriter
	if err := command.Start(); err != nil {
		cleanupAll()
		return nil, nil, fmt.Errorf("start api: %w", err)
	}
	killAPI := func() {
		_ = command.Process.Kill()
		_, _ = command.Process.Wait()
		lastLogs = logWriter.bytes()
	}

	env := &Env{
		BaseURL:     "http://127.0.0.1:" + strconv.Itoa(port),
		APIKey:      apiKey,
		ScrapersKey: scrapersKey,
	}

	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(90 * time.Second)
	for {
		response, err := client.Get(env.BaseURL + "/ready")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				break
			}
		}
		if time.Now().After(deadline) {
			killAPI()
			cleanupAll()
			if err != nil {
				return nil, nil, fmt.Errorf("api não ficou pronta: %w\n%s", err, logWriter.String())
			}
			return nil, nil, fmt.Errorf("api não ficou pronta\n%s", logWriter.String())
		}
		time.Sleep(500 * time.Millisecond)
	}

	return env, func() { killAPI(); cleanupAll() }, nil
}

func moduleRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// ringBuffer guarda os últimos bytes de log do processo filho.
type ringBuffer struct {
	data  []byte
	limit int
}

func (buffer *ringBuffer) Write(chunk []byte) (int, error) {
	buffer.data = append(buffer.data, chunk...)
	if len(buffer.data) > buffer.limit {
		buffer.data = buffer.data[len(buffer.data)-buffer.limit:]
	}
	return len(chunk), nil
}

func (buffer *ringBuffer) String() string {
	return string(buffer.data)
}

func (buffer *ringBuffer) bytes() []byte {
	return buffer.data
}
