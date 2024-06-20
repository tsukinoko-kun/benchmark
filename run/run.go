package run

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func Run(code []byte, language string) ([]byte, error) {
	switch language {
	case "java":
		return runJava(code)
	case "go":
		return runGo(code)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
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

var (
	goDockerfile = []byte(`FROM golang:alpine
RUN apk add --no-cache hyperfine
COPY . /app
WORKDIR /app
RUN go mod init github.com/tsukinoko-kun/benchmark
RUN go build -v -o main main.go
CMD /app/main \
	&& hyperfine "/app/main" -i --warmup 8 -N 2> /dev/null \
`)

	javaDockerfile = []byte(`FROM amazoncorretto:22-alpine-jdk
RUN apk add --no-cache hyperfine
COPY . /app
WORKDIR /app
RUN java Main.java
CMD java Main.java \
	&& hyperfine "java Main.java" -i --warmup 4 -N 2> /dev/null \
	&& java -XX:StartFlightRecording:filename=recording.jfr -XX:FlightRecorderOptions:stackdepth=256 -XX:StartFlightRecording:method-profiling=max Main.java > /dev/null \
	&& jfr view allocation-by-class recording.jfr \
	&& jfr view gc recording.jfr \
`)
)

func runJava(code []byte) ([]byte, error) {
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
	cmd := exec.Command("docker", "build", "-t", id, ".")
	cmd.Dir = projDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return out, errors.Join(errors.New("failed to build Docker image"), err)
	}

	// Run the Docker container
	cmd = exec.Command("docker", "run", "--rm", id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, errors.Join(errors.New("failed to run Docker container"), errors.New(string(out)), err)
	}

	// Delete the Docker image
	cmd = exec.Command("docker", "rmi", id)
	if outRmi, err := cmd.CombinedOutput(); err != nil {
		return out, errors.Join(errors.New("failed to delete Docker image"), errors.New(string(outRmi)), err)
	}

	return out, nil
}

func runGo(code []byte) ([]byte, error) {
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
	cmd := exec.Command("docker", "build", "-t", id, ".")
	cmd.Dir = projDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return out, errors.Join(errors.New("failed to build Docker image"), err)
	}

	// Run the Docker container
	cmd = exec.Command("docker", "run", "--rm", id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, errors.Join(errors.New("failed to run Docker container"), errors.New(string(out)), err)
	}

	// Delete the Docker image
	cmd = exec.Command("docker", "rmi", id)
	if outRmi, err := cmd.CombinedOutput(); err != nil {
		return out, errors.Join(errors.New("failed to delete Docker image"), errors.New(string(outRmi)), err)
	}

	return out, nil
}
