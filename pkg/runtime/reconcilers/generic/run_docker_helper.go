// pkg/reconciler/run_docker_helper.go
package generic

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type DockerBuildResult struct {
	Error string
}

// executeDockerBuild builds a container image using the specified OCI builder.
//
// builder selection order:
//  1. spec.Builder field (explicit — overrides everything)
//  2. OCI_BUILDER environment variable (cluster-wide default)
//  3. "docker" (fallback)
//
// Supported builders:
//
//	"docker"  — docker build -f <dockerfile> -t <image> <workingDir>
//	"podman"  — podman build -f <dockerfile> -t <image> <workingDir>
//	"buildah" — buildah bud -f <dockerfile> -t <image> <workingDir>
//	"kaniko"  — /kaniko/executor --dockerfile --context dir:// --destination
//	            kaniko builds without a Docker socket — works in any Kubernetes pod.
func executeDockerBuild(ctx context.Context, workingDir, dockerfile, image, builder string) DockerBuildResult {
	if builder == "" {
		builder = os.Getenv("OCI_BUILDER")
	}
	if builder == "" {
		builder = "docker"
	}

	var cmd *exec.Cmd
	switch builder {
	case "kaniko":
		// kaniko builds without a Docker socket — works in any Kubernetes pod.
		// Expects to run as /kaniko/executor in a kaniko image.
		cmd = exec.CommandContext(ctx,
			"/kaniko/executor",
			"--dockerfile", dockerfile,
			"--context", "dir://"+workingDir,
			"--destination", image,
		)
	case "buildah":
		cmd = exec.CommandContext(ctx,
			"buildah", "bud",
			"-f", dockerfile,
			"-t", image,
			workingDir,
		)
	default: // "docker" or "podman" — same CLI interface
		cmd = exec.CommandContext(ctx,
			builder, "build",
			"-f", dockerfile,
			"-t", image,
			workingDir,
		)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return DockerBuildResult{
			Error: fmt.Sprintf("%s build: %v (%s)", builder, err, out),
		}
	}

	return DockerBuildResult{}
}

type DockerPushResult struct {
	Error string
}

// executeDockerPush pushes the image using the specified OCI builder.
// kaniko handles push during build via --destination — this function is a no-op
// for kaniko; the caller should skip push when builder == "kaniko".
func executeDockerPush(ctx context.Context, image, builder string) DockerPushResult {
	if builder == "" {
		builder = os.Getenv("OCI_BUILDER")
	}
	if builder == "" {
		builder = "docker"
	}

	// kaniko pushes during build — separate push is meaningless
	if builder == "kaniko" {
		return DockerPushResult{}
	}

	var cmd *exec.Cmd
	switch builder {
	case "buildah":
		cmd = exec.CommandContext(ctx, "buildah", "push", image)
	default: // docker, podman
		cmd = exec.CommandContext(ctx, builder, "push", image)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return DockerPushResult{
			Error: fmt.Sprintf("%s push: %v (%s)", builder, err, out),
		}
	}

	return DockerPushResult{}
}
