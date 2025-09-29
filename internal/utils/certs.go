package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GenerateCA shells out to `nebula-cert` to build a self-signed certificate authority.
func GenerateCA(commonName string, validityDays int) (certPEM, keyPEM string, err error) {
	if validityDays <= 0 {
		validityDays = 365
	}

	tmpDir, err := os.MkdirTemp("", "nebula-ca-")
	if err != nil {
		return "", "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	certPath := filepath.Join(tmpDir, "ca.crt")
	keyPath := filepath.Join(tmpDir, "ca.key")
	args := []string{
		"ca",
		"-name", commonName,
		"-duration", durationDaysArg(validityDays),
		"-out-crt", certPath,
		"-out-key", keyPath,
	}

	if err := runNebulaCert(args...); err != nil {
		return "", "", err
	}

	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return "", "", fmt.Errorf("read ca cert: %w", err)
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("read ca key: %w", err)
	}
	return string(certBytes), string(keyBytes), nil
}

// GenerateNodeCertificate asks `nebula-cert` to sign a node certificate with the provided CA.

func GenerateNodeCertificate(caCertPEM, caKeyPEM, commonName string, ip string, validityDays int) (certPEM, keyPEM string, err error) {

	if !strings.Contains(caCertPEM, "NEBULA CERTIFICATE") {
		return "", "", errors.New("CA certificate不是 Nebula 格式，请重新生成 CA")
	}
	if !strings.Contains(caKeyPEM, "NEBULA ") {
		return "", "", errors.New("CA 私钥不是 Nebula 格式，请重新生成 CA")
	}

	ipCIDR := ensureCIDR(ip)

	tmpDir, err := os.MkdirTemp("", "nebula-node-")
	if err != nil {
		return "", "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	caCertPath := filepath.Join(tmpDir, "ca.crt")
	caKeyPath := filepath.Join(tmpDir, "ca.key")
	if err := os.WriteFile(caCertPath, []byte(caCertPEM), 0o600); err != nil {
		return "", "", fmt.Errorf("write ca cert: %w", err)
	}
	if err := os.WriteFile(caKeyPath, []byte(caKeyPEM), 0o600); err != nil {
		return "", "", fmt.Errorf("write ca key: %w", err)
	}

	certPath := filepath.Join(tmpDir, "node.crt")
	keyPath := filepath.Join(tmpDir, "node.key")

	baseArgs := []string{
		"sign",
		"-ca-crt", caCertPath,
		"-ca-key", caKeyPath,
		"-name", commonName,
		"-ip", ipCIDR,
		"-out-crt", certPath,
		"-out-key", keyPath,
	}

	duration := durationDaysArg(validityDays)
	args := append(baseArgs, "-duration", duration)

	if err := runNebulaCert(args...); err != nil {
		if strings.Contains(err.Error(), "root certificate constraints") || strings.Contains(err.Error(), "certificate expires after") {
			if err := runNebulaCert(baseArgs...); err != nil {
				return "", "", err
			}
		} else {
			return "", "", err
		}
	}

	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return "", "", fmt.Errorf("read node cert: %w", err)
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("read node key: %w", err)
	}
	return string(certBytes), string(keyBytes), nil
}

func durationDaysArg(days int) string {
	if days <= 0 {
		days = 365
	}
	return fmt.Sprintf("%dh", days*24)
}

func ensureCIDR(ip string) string {
	if strings.Contains(ip, "/") {
		return ip
	}
	return fmt.Sprintf("%s/32", strings.TrimSpace(ip))
}

func runNebulaCert(args ...string) error {
	cmd := exec.Command("nebula-cert", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg != "" {
			return fmt.Errorf("nebula-cert %s 失败: %s", strings.Join(args, " "), msg)
		}
		return fmt.Errorf("nebula-cert %s 失败: %w", strings.Join(args, " "), err)
	}
	return nil
}

// CertificatesBundle is a convenience container for generated artifacts.
type CertificatesBundle struct {
	Certificate string
	PrivateKey  string
}
