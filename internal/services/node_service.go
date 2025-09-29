package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gorm.io/gorm"

	"nebula_manager/internal/models"
	"nebula_manager/internal/utils"
)

const (
	nodeProxyModeIPv4 = "ipv4"
	nodeProxyModeIPv6 = "ipv6"
)

const (
	proxyPrefixIPv4 = "https://proxy.529851.xyz/"
	proxyPrefixIPv6 = "https://proxy.529851.xyz/"
)

// NodeService manages Nebula nodes and their generated artifacts.
type NodeService struct {
	db              *gorm.DB
	caService       *CAService
	templateService *TemplateService
	settingsService *SettingsService
	dataDir         string
	apiBaseURL      string
	nebulaVersion   string
	nebulaBaseURL   string
	nebulaProxyPref string
	staticToken     string
}

// NewNodeService constructs a NodeService.
func NewNodeService(db *gorm.DB, caSvc *CAService, tplSvc *TemplateService, settingsSvc *SettingsService, dataDir string, apiBaseURL string, nebulaVersion string, nebulaBaseURL string, nebulaProxyPrefix string, staticToken string) *NodeService {
	return &NodeService{
		db:              db,
		caService:       caSvc,
		templateService: tplSvc,
		settingsService: settingsSvc,
		dataDir:         dataDir,
		apiBaseURL:      apiBaseURL,
		nebulaVersion:   nebulaVersion,
		nebulaBaseURL:   strings.TrimRight(nebulaBaseURL, "/"),
		nebulaProxyPref: nebulaProxyPrefix,
		staticToken:     staticToken,
	}
}

// CreateNodeRequest captures payload data for node creation.
type CreateNodeRequest struct {
	Name      string   `json:"name" binding:"required"`
	Role      string   `json:"role" binding:"required"`
	SubnetIP  string   `json:"subnet_ip" binding:"required"`
	PublicIP  string   `json:"public_ip"`
	Port      int      `json:"port"`
	Tags      []string `json:"tags"`
	ProxyMode string   `json:"proxy_mode"`
}

// NodeDTO is returned to API consumers.
type NodeDTO struct {
	ID             uint     `json:"id"`
	Name           string   `json:"name"`
	Role           string   `json:"role"`
	SubnetIP       string   `json:"subnet_ip"`
	SubnetHost     string   `json:"subnet_host,omitempty"`
	PublicIP       string   `json:"public_ip"`
	Port           int      `json:"port"`
	Tags           []string `json:"tags"`
	ProxyMode      string   `json:"proxy_mode"`
	InstallCommand string   `json:"install_command"`
	CreatedAt      string   `json:"created_at"`
}

// List returns all stored nodes.
func (s *NodeService) List() ([]NodeDTO, error) {
	var nodes []models.Node
	if err := s.db.Order("created_at desc").Find(&nodes).Error; err != nil {
		return nil, err
	}
	res := make([]NodeDTO, 0, len(nodes))
	for _, n := range nodes {
		res = append(res, s.toNodeDTO(n))
	}
	return res, nil
}

// Create provisions a new node and persists generated config.
func (s *NodeService) Create(req CreateNodeRequest) (*NodeDTO, error) {
	if req.Role != models.NodeRoleLighthouse && req.Role != models.NodeRoleStandard {
		return nil, fmt.Errorf("unsupported role %s", req.Role)
	}

	ca, err := s.caService.GetCA()
	if err != nil {
		return nil, err
	}
	if ca == nil {
		return nil, errors.New("CA not generated yet")
	}

	settings, err := s.settingsService.Get()
	if err != nil {
		return nil, err
	}
	subnetCIDR, subnetHost, err := s.normalizeSubnetInput(req.SubnetIP, settings)
	if err != nil {
		return nil, err
	}

	listenPort := req.Port
	if listenPort == 0 {
		listenPort = settings.HandshakePort
	}

	validity := settings.CertificateValidity
	cert, key, err := utils.GenerateNodeCertificate(ca.CertificatePEM, ca.PrivateKeyPEM, req.Name, subnetCIDR, validity)
	if err != nil {
		return nil, err
	}

	proxyMode := normalizeProxyMode(req.ProxyMode)

	tpl, err := s.templateService.EnsureDefault()
	if err != nil {
		return nil, err
	}

	lighthouses, err := s.listLighthouseNodes()
	if err != nil {
		return nil, err
	}
	if req.Role == models.NodeRoleLighthouse {
		publicHost := ""
		if req.PublicIP != "" {
			publicHost = fmt.Sprintf("%s:%d", req.PublicIP, listenPort)
		}
		lighthouses = append(lighthouses, map[string]any{
			"Name":       req.Name,
			"PublicIP":   req.PublicIP,
			"SubnetIP":   subnetHost,
			"Port":       listenPort,
			"PublicHost": publicHost,
		})
	}

	data := map[string]any{
		"Name":         req.Name,
		"CACertPath":   "ca.crt",
		"CertPath":     fmt.Sprintf("%s.crt", req.Name),
		"KeyPath":      fmt.Sprintf("%s.key", req.Name),
		"SubnetIP":     subnetHost,
		"SubnetCIDR":   subnetCIDR,
		"PublicIP":     req.PublicIP,
		"ListenPort":   listenPort,
		"IsLighthouse": req.Role == models.NodeRoleLighthouse,
		"Lighthouses":  lighthouses,
		"DeviceID":     req.Name,
	}

	rendered, err := renderTemplate(tpl.Content, data)
	if err != nil {
		return nil, err
	}

	node := &models.Node{
		Name:              req.Name,
		Role:              req.Role,
		SubnetIP:          subnetHost,
		SubnetCIDR:        subnetCIDR,
		SubnetHost:        subnetHost,
		PublicIP:          req.PublicIP,
		Port:              listenPort,
		Tags:              strings.Join(req.Tags, ","),
		DownloadProxyMode: proxyMode,
		CertificatePEM:    cert,
		PrivateKeyPEM:     key,
		ConfigContent:     rendered,
	}

	if err := s.db.Create(node).Error; err != nil {
		return nil, err
	}

	if err := s.writeArtifacts(node, ca.CertificatePEM); err != nil {
		return nil, err
	}

	dto := s.toNodeDTO(*node)
	return &dto, nil
}

// Delete removes a node and its generated artifacts.
func (s *NodeService) Delete(id uint) error {
	node, err := s.getNode(id)
	if err != nil {
		return err
	}
	if err := s.db.Delete(&models.Node{}, id).Error; err != nil {
		return err
	}
	nodeDir := filepath.Join(s.dataDir, "nodes", node.Name)
	if err := os.RemoveAll(nodeDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetConfig returns the rendered config file for a node.
func (s *NodeService) GetConfig(id uint) (string, error) {
	node, err := s.getNode(id)
	if err != nil {
		return "", err
	}
	return node.ConfigContent, nil
}

// GetArtifacts returns node cert, key, and CA cert.
type NodeArtifacts struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
	CACert      string `json:"ca_cert"`
	Config      string `json:"config"`
}

func (s *NodeService) GetArtifacts(id uint) (*NodeArtifacts, error) {
	node, ca, err := s.refreshNode(id)
	if err != nil {
		return nil, err
	}
	return &NodeArtifacts{
		Certificate: node.CertificatePEM,
		PrivateKey:  node.PrivateKeyPEM,
		CACert:      ca.CertificatePEM,
		Config:      node.ConfigContent,
	}, nil
}

func (s *NodeService) installCommand(node models.Node) string {
	base := s.apiBaseURL
	if base == "" {
		base = "http://localhost:8080"
	}
	base = strings.TrimRight(base, "/")
	if s.staticToken != "" {
		token := s.staticToken
		return fmt.Sprintf("curl -fsSL -H \"Authorization: Bearer %s\" \"%s/api/nodes/%d/install-script\" | bash", token, base, node.ID)
	}
	return fmt.Sprintf("curl -fsSL -H \"Authorization: Bearer ${NEBULA_ACCESS_TOKEN:?missing NEBULA_ACCESS_TOKEN}\" \"%s/api/nodes/%d/install-script\" | bash", base, node.ID)
}

func (s *NodeService) proxyPrefixForNode(node *models.Node) string {
	switch node.DownloadProxyMode {
	case nodeProxyModeIPv4:
		return proxyPrefixIPv4
	case nodeProxyModeIPv6:
		return proxyPrefixIPv6
	default:
		pref := strings.TrimSpace(s.nebulaProxyPref)
		if pref == "" {
			return ""
		}
		if !strings.HasSuffix(pref, "/") {
			pref += "/"
		}
		return pref
	}
}

// GenerateInstallScript renders a shell script that installs the node artifacts on a host.

func (s *NodeService) GenerateInstallScript(id uint) (string, error) {
	node, _, err := s.refreshNode(id)
	if err != nil {
		return "", err
	}

	apiBase := s.apiBaseURL
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	apiBase = strings.TrimRight(apiBase, "/")
	nebulaBase := s.nebulaBaseURL
	if nebulaBase == "" {
		nebulaBase = "https://github.com/slackhq/nebula/releases/download"
	}
	nebulaBase = strings.TrimRight(nebulaBase, "/")
	nebulaVersion := s.nebulaVersion
	if nebulaVersion == "" {
		nebulaVersion = "1.9.3"
	}
	proxyPrefix := s.proxyPrefixForNode(node)

	var b strings.Builder
	b.WriteString("#!/bin/bash\n")
	b.WriteString("set -euo pipefail\n\n")
	b.WriteString(fmt.Sprintf("API_BASE=\"${NEBULA_MANAGER_API:-%s}\"\n", apiBase))
	b.WriteString(fmt.Sprintf("NODE_ID=%d\n", node.ID))
	b.WriteString(fmt.Sprintf("NODE_NAME=\"%s\"\n", node.Name))
	b.WriteString("NEBULA_DIR=\"${NEBULA_DIR:-/etc/nebula}\"\n")
	b.WriteString("TMP_DIR=$(mktemp -d)\n")
	b.WriteString("trap 'rm -rf \"$TMP_DIR\"' EXIT\n\n")
	b.WriteString(fmt.Sprintf("NEBULA_VERSION=\"${NEBULA_VERSION:-%s}\"\n", nebulaVersion))
	b.WriteString(fmt.Sprintf("NEBULA_DOWNLOAD_BASE=\"%s\"\n", escapeForDoubleQuotes(nebulaBase)))
	b.WriteString(fmt.Sprintf("NEBULA_PROXY_PREFIX=\"%s\"\n", escapeForDoubleQuotes(proxyPrefix)))
	if s.staticToken != "" {
		escapedToken := escapeForDoubleQuotes(s.staticToken)
		b.WriteString("NEBULA_ACCESS_TOKEN=\"" + escapedToken + "\"\n")
	} else {
		b.WriteString("NEBULA_ACCESS_TOKEN=\"${NEBULA_ACCESS_TOKEN:?missing NEBULA_ACCESS_TOKEN}\"\n")
	}
	b.WriteString("CURL_AUTH=(-H \"Authorization: Bearer $NEBULA_ACCESS_TOKEN\")\n\n")
	b.WriteString("if ! command -v curl >/dev/null 2>&1; then\n")
	b.WriteString("  echo '需要安装 curl 用于下载文件' >&2\n")
	b.WriteString("  exit 1\n")
	b.WriteString("fi\n")
	b.WriteString("if ! command -v tar >/dev/null 2>&1; then\n")
	b.WriteString("  echo '需要安装 tar 用于解压归档' >&2\n")
	b.WriteString("  exit 1\n")
	b.WriteString("fi\n\n")
	b.WriteString("if ! command -v systemctl >/dev/null 2>&1; then\n")
	b.WriteString("  echo '当前系统缺少 systemctl，无法自动创建 systemd 服务' >&2\n")
	b.WriteString("  exit 1\n")
	b.WriteString("fi\n\n")
	b.WriteString("OS=$(uname -s | tr 'A-Z' 'a-z')\n")
	b.WriteString("ARCH=$(uname -m)\n")
	b.WriteString("if [ \"$OS\" != \"linux\" ]; then\n")
	b.WriteString("  echo '当前安装脚本仅支持 Linux 系统' >&2\n")
	b.WriteString("  exit 1\n")
	b.WriteString("fi\n")
	b.WriteString("case $ARCH in\n")
	b.WriteString("  x86_64|amd64) ARCH=amd64 ;;\n")
	b.WriteString("  aarch64|arm64) ARCH=arm64 ;;\n")
	b.WriteString("  armv7l|armv7) ARCH=arm ;;\n")
	b.WriteString("  armv6l) ARCH=arm6 ;;\n")
	b.WriteString("  i386|i686) ARCH=386 ;;\n")
	b.WriteString("  *) echo \"暂不支持的 CPU 架构: $ARCH\" >&2; exit 1 ;;\n")
	b.WriteString("esac\n")
	b.WriteString("NEBULA_PACKAGE=\"nebula-linux-$ARCH.tar.gz\"\n")
	b.WriteString("BASE_URL=\"$NEBULA_DOWNLOAD_BASE/v$NEBULA_VERSION/$NEBULA_PACKAGE\"\n")
	b.WriteString("if [ -n \"$NEBULA_PROXY_PREFIX\" ]; then\n")
	b.WriteString("  DOWNLOAD_URL=\"$NEBULA_PROXY_PREFIX$BASE_URL\"\n")
	b.WriteString("else\n")
	b.WriteString("  DOWNLOAD_URL=\"$BASE_URL\"\n")
	b.WriteString("fi\n\n")
	b.WriteString("echo \"从 $DOWNLOAD_URL 下载 Nebula 二进制...\"\n")
	b.WriteString("curl -fsSL \"$DOWNLOAD_URL\" -o \"$TMP_DIR/$NEBULA_PACKAGE\"\n")
	b.WriteString("tar -xzf \"$TMP_DIR/$NEBULA_PACKAGE\" -C \"$TMP_DIR\" nebula\n")
	b.WriteString("sudo install -m 755 \"$TMP_DIR/nebula\" /usr/local/bin/nebula\n\n")
	b.WriteString("echo \"从 $API_BASE 获取节点归档...\"\n")
	b.WriteString("curl -fsSL \"${CURL_AUTH[@]}\" \"$API_BASE/api/nodes/$NODE_ID/bundle\" -o \"$TMP_DIR/node_bundle.tar.gz\"\n")
	b.WriteString("tar -xzf \"$TMP_DIR/node_bundle.tar.gz\" -C \"$TMP_DIR\"\n\n")
	b.WriteString("sudo install -d -m 755 \"$NEBULA_DIR\"\n")
	b.WriteString("sudo install -m 600 \"$TMP_DIR/ca.crt\" \"$NEBULA_DIR/ca.crt\"\n")
	b.WriteString(fmt.Sprintf("sudo install -m 600 \"$TMP_DIR/%s.crt\" \"$NEBULA_DIR/%s.crt\"\n", node.Name, node.Name))
	b.WriteString(fmt.Sprintf("sudo install -m 600 \"$TMP_DIR/%s.key\" \"$NEBULA_DIR/%s.key\"\n", node.Name, node.Name))
	b.WriteString("sudo install -m 640 \"$TMP_DIR/config.yml\" \"$NEBULA_DIR/config.yml\"\n")
	b.WriteString("sudo chmod 600 \"$NEBULA_DIR\"/*.key\n")
	b.WriteString("sudo tee /etc/systemd/system/nebula.service >/dev/null <<'UNIT'\n")
	b.WriteString("[Unit]\n")
	b.WriteString("Description=Nebula VPN 节点\n")
	b.WriteString("After=network-online.target\n")
	b.WriteString("Wants=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("ExecStart=/usr/local/bin/nebula -config /etc/nebula/config.yml\n")
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=5\n")
	b.WriteString("User=root\n")
	b.WriteString("WorkingDirectory=/etc/nebula\n")
	b.WriteString("LimitNOFILE=65535\n\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")
	b.WriteString("UNIT\n")
	b.WriteString("sudo systemctl daemon-reload\n")
	b.WriteString("sudo systemctl enable --now nebula.service\n")
	b.WriteString("echo \"Nebula 节点已部署并以 systemd 服务运行\"\n")
	b.WriteString("sudo systemctl status nebula.service --no-pager\n")

	return b.String(), nil
}

// BuildBundle returns a tar.gz archive with the node's certificate, key, CA, and config files.
func (s *NodeService) BuildBundle(id uint) ([]byte, error) {
	node, ca, err := s.refreshNode(id)
	if err != nil {
		return nil, err
	}
	artifacts := &NodeArtifacts{
		Certificate: node.CertificatePEM,
		PrivateKey:  node.PrivateKeyPEM,
		CACert:      ca.CertificatePEM,
		Config:      node.ConfigContent,
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	add := func(name string, mode int64, content string) error {
		header := &tar.Header{
			Name:    name,
			Mode:    mode,
			Size:    int64(len(content)),
			ModTime: time.Now(),
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		_, err := tw.Write([]byte(content))
		return err
	}

	if err := add("ca.crt", 0o600, artifacts.CACert); err != nil {
		tw.Close()
		gz.Close()
		return nil, err
	}
	if err := add(fmt.Sprintf("%s.crt", node.Name), 0o600, artifacts.Certificate); err != nil {
		tw.Close()
		gz.Close()
		return nil, err
	}
	if err := add(fmt.Sprintf("%s.key", node.Name), 0o600, artifacts.PrivateKey); err != nil {
		tw.Close()
		gz.Close()
		return nil, err
	}
	if err := add("config.yml", 0o640, artifacts.Config); err != nil {
		tw.Close()
		gz.Close()
		return nil, err
	}

	if err := tw.Close(); err != nil {
		gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *NodeService) refreshNode(id uint) (*models.Node, *models.CA, error) {
	node, err := s.getNode(id)
	if err != nil {
		return nil, nil, err
	}
	ca, err := s.caService.GetCA()
	if err != nil {
		return nil, nil, err
	}
	if ca == nil {
		return nil, nil, errors.New("no CA present")
	}
	if err := s.regenerateNodeArtifacts(node, ca); err != nil {
		return nil, nil, err
	}
	return node, ca, nil
}

func (s *NodeService) regenerateNodeArtifacts(node *models.Node, ca *models.CA) error {
	settings, err := s.settingsService.Get()
	if err != nil {
		return err
	}
	if err := s.ensureNodeSubnet(node, settings); err != nil {
		return err
	}
	validity := 365
	if settings != nil && settings.CertificateValidity > 0 {
		validity = settings.CertificateValidity
	}

	listenPort := node.Port
	if settings != nil && settings.HandshakePort != 0 {
		if listenPort == 0 {
			listenPort = settings.HandshakePort
		}
	}
	if listenPort == 0 {
		listenPort = 4242
	}
	if node.Port != listenPort {
		node.Port = listenPort
	}

	cert, key, err := utils.GenerateNodeCertificate(ca.CertificatePEM, ca.PrivateKeyPEM, node.Name, node.SubnetCIDR, validity)
	if err != nil {
		return err
	}

	tpl, err := s.templateService.EnsureDefault()
	if err != nil {
		return err
	}
	lighthouses, err := s.listLighthouseNodes()
	if err != nil {
		return err
	}

	data := map[string]any{
		"Name":         node.Name,
		"CACertPath":   "ca.crt",
		"CertPath":     fmt.Sprintf("%s.crt", node.Name),
		"KeyPath":      fmt.Sprintf("%s.key", node.Name),
		"SubnetIP":     node.SubnetHost,
		"SubnetCIDR":   node.SubnetCIDR,
		"PublicIP":     node.PublicIP,
		"ListenPort":   listenPort,
		"IsLighthouse": node.Role == models.NodeRoleLighthouse,
		"Lighthouses":  lighthouses,
		"DeviceID":     node.Name,
	}

	rendered, err := renderTemplate(tpl.Content, data)
	if err != nil {
		return err
	}

	node.CertificatePEM = cert
	node.PrivateKeyPEM = key
	node.ConfigContent = rendered

	if err := s.db.Save(node).Error; err != nil {
		return err
	}

	if err := s.writeArtifacts(node, ca.CertificatePEM); err != nil {
		return err
	}

	return nil
}

func escapeForDoubleQuotes(val string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
	return replacer.Replace(val)
}

func (s *NodeService) getNode(id uint) (*models.Node, error) {
	var node models.Node
	if err := s.db.First(&node, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("node %d not found", id)
		}
		return nil, err
	}
	return &node, nil
}

func (s *NodeService) listLighthouseNodes() ([]map[string]any, error) {
	var nodes []models.Node
	if err := s.db.Where("role = ?", models.NodeRoleLighthouse).Find(&nodes).Error; err != nil {
		return nil, err
	}
	settings, err := s.settingsService.Get()
	if err != nil {
		return nil, err
	}
	defaultPort := 4242
	if settings != nil && settings.HandshakePort != 0 {
		defaultPort = settings.HandshakePort
	}
	lighthouses := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		port := n.Port
		if port == 0 {
			port = defaultPort
		}
		subnetHost := n.SubnetHost
		if subnetHost == "" {
			subnetHost = n.SubnetIP
		}
		publicHost := ""
		if n.PublicIP != "" {
			publicHost = fmt.Sprintf("%s:%d", n.PublicIP, port)
		}
		lighthouses = append(lighthouses, map[string]any{
			"Name":       n.Name,
			"PublicIP":   n.PublicIP,
			"SubnetIP":   subnetHost,
			"Port":       port,
			"PublicHost": publicHost,
		})
	}
	return lighthouses, nil
}

func (s *NodeService) writeArtifacts(node *models.Node, caCert string) error {
	nodeDir := filepath.Join(s.dataDir, "nodes", node.Name)
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		return err
	}

	files := map[string]string{
		"ca.crt":                         caCert,
		fmt.Sprintf("%s.crt", node.Name): node.CertificatePEM,
		fmt.Sprintf("%s.key", node.Name): node.PrivateKeyPEM,
		"config.yml":                     node.ConfigContent,
	}

	for name, content := range files {
		fullPath := filepath.Join(nodeDir, name)
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func renderTemplate(tpl string, data map[string]any) (string, error) {
	parsed, err := template.New("node").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := parsed.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *NodeService) toNodeDTO(node models.Node) NodeDTO {
	var tags []string
	if node.Tags != "" {
		tags = strings.Split(node.Tags, ",")
	}
	subnetCIDR := node.SubnetCIDR
	subnetHost := node.SubnetHost
	fallback := node.SubnetCIDR
	if fallback == "" {
		fallback = node.SubnetIP
	}
	if (subnetCIDR == "" || subnetHost == "") && fallback != "" {
		if cidr, host, err := s.normalizeSubnetInput(fallback, nil); err == nil {
			if subnetCIDR == "" {
				subnetCIDR = cidr
			}
			if subnetHost == "" {
				subnetHost = host
			}
		}
	}
	if subnetHost == "" {
		subnetHost = node.SubnetIP
	}
	if subnetCIDR == "" {
		subnetCIDR = subnetHost
	}
	return NodeDTO{
		ID:             node.ID,
		Name:           node.Name,
		Role:           node.Role,
		SubnetIP:       subnetCIDR,
		SubnetHost:     subnetHost,
		PublicIP:       node.PublicIP,
		Port:           node.Port,
		Tags:           tags,
		ProxyMode:      node.DownloadProxyMode,
		InstallCommand: s.installCommand(node),
		CreatedAt:      node.CreatedAt.Format(time.RFC3339),
	}
}

func normalizeProxyMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case nodeProxyModeIPv4:
		return nodeProxyModeIPv4
	case nodeProxyModeIPv6:
		return nodeProxyModeIPv6
	default:
		return ""
	}
}

func (s *NodeService) normalizeSubnetInput(input string, settings *models.NetworkSetting) (string, string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", "", errors.New("subnet ip required")
	}
	host := trimmed
	maskPart := ""
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		host = strings.TrimSpace(trimmed[:idx])
		maskPart = strings.TrimSpace(trimmed[idx+1:])
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", "", fmt.Errorf("invalid subnet ip: %s", host)
	}
	if maskPart == "" {
		maskPart = defaultMaskForIP(ip, settings)
	}
	maskInt, err := strconv.Atoi(maskPart)
	if err != nil {
		return "", "", fmt.Errorf("invalid subnet mask: %s", maskPart)
	}
	if ip.To4() != nil {
		if maskInt < 0 || maskInt > 32 {
			return "", "", fmt.Errorf("invalid subnet mask: %d", maskInt)
		}
	} else {
		if maskInt < 0 || maskInt > 128 {
			return "", "", fmt.Errorf("invalid subnet mask: %d", maskInt)
		}
	}
	cidr := fmt.Sprintf("%s/%d", host, maskInt)
	return cidr, host, nil
}

func (s *NodeService) ensureNodeSubnet(node *models.Node, settings *models.NetworkSetting) error {
	if node.SubnetHost != "" && node.SubnetCIDR != "" {
		node.SubnetIP = node.SubnetHost
		return nil
	}
	base := node.SubnetCIDR
	if base == "" {
		base = node.SubnetIP
	}
	cidr, host, err := s.normalizeSubnetInput(base, settings)
	if err != nil {
		return err
	}
	node.SubnetCIDR = cidr
	node.SubnetHost = host
	node.SubnetIP = host
	if node.ID == 0 {
		return nil
	}
	return s.db.Model(node).Updates(map[string]any{
		"subnet_ip":     host,
		"subnet_c_id_r": cidr,
		"subnet_host":   host,
	}).Error
}

func defaultMaskForIP(ip net.IP, settings *models.NetworkSetting) string {
	if settings != nil {
		if cidr := strings.TrimSpace(settings.DefaultSubnet); cidr != "" {
			if _, network, err := net.ParseCIDR(cidr); err == nil {
				if ones, _ := network.Mask.Size(); ones > 0 {
					return strconv.Itoa(ones)
				}
			}
		}
	}
	if ip.To4() != nil {
		return "24"
	}
	return "64"
}
