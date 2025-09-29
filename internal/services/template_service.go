package services

import (
	"errors"

	"gorm.io/gorm"

	"nebula_manager/internal/models"
)

const defaultTemplateName = "default"

const defaultTemplateContent = `pki:
  ca: {{ .CACertPath }}
  cert: {{ .CertPath }}
  key: {{ .KeyPath }}
static_host_map:
{{- range .Lighthouses }}
{{- if .PublicHost }}
  "{{ .PublicHost }}": ["{{ .SubnetIP }}"]
{{- end }}
{{- end }}
lighthouse:
  am_lighthouse: {{ .IsLighthouse }}
  interval: 60
  hosts:
{{- range .Lighthouses }}
    - "{{ .SubnetIP }}"
{{- end }}
listen:
  host: 0.0.0.0
  port: {{ .ListenPort }}
  batch: 64
punchy:
  punch: true
tun:
  dev: nebula{{ if .DeviceID }}{{ .DeviceID }}{{ end }}
  mtu: 1300
  unsafe_routes: []
  cipher: aes
  drop_local_broadcast: false
  drop_multicast: false
  disabled: false
  tx_queue: 5000
  ip: {{ .SubnetCIDR }}
firewall:
  conntrack:
    tcp_timeout: 12m
    udp_timeout: 3m
    default_allow: false
  outbound:
    - port: any
      proto: any
      host: any
  inbound:
    - port: any
      proto: any
      host: any
`

const legacyDefaultTemplateContent = `pki:
  ca: {{ .CACertPath }}
  cert: {{ .CertPath }}
  key: {{ .KeyPath }}
static_host_map:
{{- range .Lighthouses }}
  "{{ .SubnetIP }}": ["{{ .PublicHost }}"]
{{- end }}
lighthouse:
  am_lighthouse: {{ .IsLighthouse }}
  interval: 60
  hosts:
{{- range .Lighthouses }}
    - "{{ .SubnetIP }}"
{{- end }}
listen:
  host: 0.0.0.0
  port: {{ .ListenPort }}
  batch: 64
punchy:
  punch: true
tun:
  dev: nebula{{ if .DeviceID }}{{ .DeviceID }}{{ end }}
  mtu: 1300
  unsafe_routes: []
  cipher: aes
  drop_local_broadcast: false
  drop_multicast: false
  disabled: false
  tx_queue: 5000
  ip: {{ .SubnetCIDR }}
firewall:
  conntrack:
    tcp_timeout: 12m
    udp_timeout: 3m
    default_allow: false
  outbound:
    - port: any
      proto: any
      host: any
  inbound:
    - port: any
      proto: any
      host: any
`

// TemplateService manages configuration templates stored in the database.
type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService constructs a TemplateService.
func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{db: db}
}

// EnsureDefault returns the default template, creating it if missing.
func (s *TemplateService) EnsureDefault() (*models.ConfigTemplate, error) {
	tpl, err := s.GetByName(defaultTemplateName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tpl = &models.ConfigTemplate{
				Name:    defaultTemplateName,
				Content: defaultTemplateContent,
			}
			if err := s.db.Create(tpl).Error; err != nil {
				return nil, err
			}
			return tpl, nil
		}
		return nil, err
	}
	if tpl.Content == legacyDefaultTemplateContent {
		tpl.Content = defaultTemplateContent
		if err := s.db.Save(tpl).Error; err != nil {
			return nil, err
		}
	}
	return tpl, nil
}

// GetByName retrieves a template by name.
func (s *TemplateService) GetByName(name string) (*models.ConfigTemplate, error) {
	var tpl models.ConfigTemplate
	if err := s.db.Where("name = ?", name).First(&tpl).Error; err != nil {
		return nil, err
	}
	return &tpl, nil
}

// List returns all templates.
func (s *TemplateService) List() ([]models.ConfigTemplate, error) {
	var templates []models.ConfigTemplate
	if err := s.db.Order("name").Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// Upsert creates or updates a template.
func (s *TemplateService) Upsert(tpl *models.ConfigTemplate) error {
	existing, err := s.GetByName(tpl.Name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.db.Create(tpl).Error
		}
		return err
	}
	existing.Content = tpl.Content
	return s.db.Save(existing).Error
}

// Delete removes a template by ID.
func (s *TemplateService) Delete(id uint) error {
	return s.db.Delete(&models.ConfigTemplate{}, id).Error
}
