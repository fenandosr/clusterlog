package clusterlog

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func gitOutput(root string, args ...string) (string, error) {
	full := append([]string{"-C", root}, args...)
	cmd := exec.Command("git", full...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}
	return strings.TrimSpace(string(output)), nil
}

func isGitRepo(root string) bool {
	output, err := gitOutput(root, "rev-parse", "--is-inside-work-tree")
	return err == nil && output == "true"
}

func (a *App) commitPaths(root string, identity Identity, message string, paths []string) (string, error) {
	if !a.Options.Commit {
		return "", nil
	}
	if !isGitRepo(root) {
		return "", NewError(ExitGit, "not_git_repo", "--commit requiere que el proyecto sea un repositorio Git", root)
	}
	rel, err := relativePaths(root, paths)
	if err != nil {
		return "", err
	}
	addArgs := append([]string{"add", "--"}, rel...)
	if _, err := gitOutput(root, addArgs...); err != nil {
		return "", NewError(ExitGit, "git_add_failed", err.Error(), rel)
	}

	args := []string{
		"-c", "user.name=" + identity.Admin.Name,
		"-c", "user.email=" + identity.Admin.Email,
		"-c", "gpg.format=ssh",
		"-c", "user.signingkey=" + identity.SigningKey,
		"commit", "-S", "-m", message, "--",
	}
	args = append(args, rel...)
	if _, err := gitOutput(root, args...); err != nil {
		return "", NewError(ExitGit, "git_commit_failed",
			"no se pudo crear el commit SSH firmado", map[string]any{
				"cause":       err.Error(),
				"signing_key": identity.SigningKey,
				"hint":        "compruebe OpenSSH/ssh-keygen y que la llave privada esté disponible o cargada en ssh-agent",
			})
	}
	sha, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return sha, nil
}

func verifyCommit(root, sha string) error {
	if sha == "" {
		sha = "HEAD"
	}
	allowed := filepath.Join(root, "config", "allowed_signers")
	_, err := gitOutput(root,
		"-c", "gpg.format=ssh",
		"-c", "gpg.ssh.allowedSignersFile="+allowed,
		"verify-commit", sha,
	)
	if err != nil {
		return NewError(ExitValidation, "invalid_commit_signature", "el commit no tiene una firma SSH válida y autorizada", map[string]any{
			"commit": sha,
			"cause":  err.Error(),
		})
	}
	return nil
}
