package run

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	timeout = 30 * time.Second
	maxOut  = 1024 * 4
)

func Run(code []byte, language string) ([]byte, error) {
	// use context to abort the process if it takes too long
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var output []byte
	var err error

	switch language {
	case "java":
		output, err = runJava(ctx, code)
	case "go":
		output, err = runGo(ctx, code)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return output, errors.Join(fmt.Errorf("execution timed out after %s", timeout), errors.New(string(output)))
	}

	return output, err
}

func projectId() string {
	return fmt.Sprintf("benchmark-%s-%d", randomString(6), time.Now().Unix())
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func init() {
	go func() {
		cmd := exec.Command("docker", "pull", "amazoncorretto:22-alpine-jdk")
		if out, err := cmd.CombinedOutput(); err != nil {
			panic(errors.Join(errors.New("failed to pull Docker image"), errors.New(string(out)), err))
		}
	}()

	go func() {
		cmd := exec.Command("docker", "pull", "golang:alpine")
		if out, err := cmd.CombinedOutput(); err != nil {
			panic(errors.Join(errors.New("failed to pull Docker image"), errors.New(string(out)), err))
		}
	}()
}

var (
	goDockerfile = []byte(`FROM golang:alpine
RUN apk add --no-cache hyperfine
COPY . /app
WORKDIR /app
RUN go mod init github.com/tsukinoko-kun/benchmark
RUN go mod tidy
CMD go build -v -o main main.go \
	&& /app/main \
	&& echo "---" \
	&& hyperfine "/app/main" -i --warmup 8 -N 2> /dev/null \
`)

	javaDockerfile = []byte(`FROM amazoncorretto:22-alpine-jdk
RUN apk add --no-cache hyperfine
COPY . /app
WORKDIR /app
CMD java Main.java \
	&& echo "---" \
	&& hyperfine "java Main.java" -i --warmup 4 -N 2> /dev/null \
	&& java -XX:StartFlightRecording:filename=recording.jfr -XX:FlightRecorderOptions:stackdepth=256 -XX:StartFlightRecording:method-profiling=max Main.java > /dev/null \
	&& jfr view allocation-by-class recording.jfr \
	&& jfr view gc recording.jfr \
`)
)

func runJava(ctx context.Context, code []byte) ([]byte, error) {
	id := "java-" + projectId()
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.Join(errors.New("failed to get working directory"), err)
	}
	projDir := filepath.Join(wd, id)

	// Create a project directory
	if err := os.MkdirAll(projDir, 0755); err != nil {
		return nil, errors.Join(errors.New("failed to create project directory"), err)
	}
	defer os.RemoveAll(projDir)

	// Write the Java source file
	if err := os.WriteFile(filepath.Join(projDir, "Main.java"), code, 0644); err != nil {
		return nil, errors.Join(errors.New("failed to write Java source file"), err)
	}

	// Write the Dockerfile
	if err := os.WriteFile(filepath.Join(projDir, "Dockerfile"), javaDockerfile, 0644); err != nil {
		return nil, errors.Join(errors.New("failed to write Dockerfile"), err)
	}

	// Build the Docker image
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", id, ".")
	cmd.Dir = projDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return out, errors.Join(errors.New("failed to build Docker image"), errors.New(string(out)), err)
	}

	defer func() {
		// Remove the Docker image
		cmd := exec.Command("docker", "rmi", id)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Println(errors.Join(errors.New("failed to remove Docker image"), errors.New(string(out)), err))
		}
	}()

	// Run the Docker container
	cmd = exec.CommandContext(ctx, "docker", "run", id)
	out, err := cmd.CombinedOutput()

	// Truncate the output
	if out != nil && len(out) > maxOut {
		out = out[:maxOut]
	}

	if err != nil {
		return out, errors.Join(errors.New("failed to run Docker container"), errors.New(string(out)), err)
	}

	return out, nil
}

func runGo(ctx context.Context, code []byte) ([]byte, error) {
	id := "go-" + projectId()
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.Join(errors.New("failed to get working directory"), err)
	}
	projDir := filepath.Join(wd, id)

	// Create a project directory
	if err := os.MkdirAll(projDir, 0755); err != nil {
		return nil, errors.Join(errors.New("failed to create project directory"), err)
	}
	defer os.RemoveAll(projDir)

	// Write the Go source file
	if err := os.WriteFile(filepath.Join(projDir, "main.go"), code, 0644); err != nil {
		return nil, errors.Join(errors.New("failed to write Go source file"), err)
	}

	// Write the Dockerfile
	if err := os.WriteFile(filepath.Join(projDir, "Dockerfile"), goDockerfile, 0644); err != nil {
		return nil, errors.Join(errors.New("failed to write Dockerfile"), err)
	}

	// Build the Docker image
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", id, ".")
	cmd.Dir = projDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return out, errors.Join(errors.New("failed to build Docker image"), err)
	}

	defer func() {
		// Remove the Docker image
		cmd := exec.Command("docker", "rmi", id)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Println(errors.Join(errors.New("failed to remove Docker image"), errors.New(string(out)), err))
		}
	}()

	// Run the Docker container
	cmd = exec.CommandContext(ctx, "docker", "run", id)
	out, err := cmd.CombinedOutput()

	// Truncate the output
	if out != nil && len(out) > maxOut {
		out = out[:maxOut]
	}

	if err != nil {
		return out, errors.Join(errors.New("failed to run Docker container"), errors.New(string(out)), err)
	}

	return out, nil
}
