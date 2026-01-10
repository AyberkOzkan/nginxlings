package main

import "path/filepath"

// LoadExercises, verilen root dizine göre tüm egzersizleri tanımlar.
func LoadExercises(baseDir string) []Exercise {
	exRoot := filepath.Join(baseDir, "exercises")

	return []Exercise{
		{
			ID:          "01_static_server",
			DefaultName: "01 - Basit Statik Nginx Sunucusu",
			Path:        filepath.Join(exRoot, "01_static_server"),
		},
		{
			ID:          "02_config_layout",
			DefaultName: "02 - Config Layout ve Default Paths",
			Path:        filepath.Join(exRoot, "02_config_layout"),
		},
		{
			ID:          "03_context_hierarchy",
			DefaultName: "03 - Context Hiyerarşisi (main/http/server/location)",
			Path:        filepath.Join(exRoot, "03_context_hierarchy"),
		},
		{
			ID:          "04_worker_user",
			DefaultName: "04 - Worker User ve İzinler",
			Path:        filepath.Join(exRoot, "04_worker_user"),
		},
		{
			ID:          "05_install_verify",
			DefaultName: "05 - Kurulum ve Doğrulama (nginx -t / -T)",
			Path:        filepath.Join(exRoot, "05_install_verify"),
		},
		{
			ID:          "06_reverse_proxy",
			DefaultName: "06 - HTTP Reverse Proxy Temelleri",
			Path:        filepath.Join(exRoot, "06_reverse_proxy"),
		},
		{
			ID:          "07_http_load_balancer",
			DefaultName: "07 - HTTP Load Balancer (Layer 7)",
			Path:        filepath.Join(exRoot, "07_http_load_balancer"),
		},
		{
			ID:          "08_stream_load_balancer",
			DefaultName: "08 - TCP/UDP Load Balancer (stream, Layer 4)",
			Path:        filepath.Join(exRoot, "08_stream_load_balancer"),
		},
		{
			ID:          "09_shared_zones_basics",
			DefaultName: "09 - Shared Memory Zones Temelleri",
			Path:        filepath.Join(exRoot, "09_shared_zones_basics"),
		},
		{
			ID:          "10_upstream_status_zone",
			DefaultName: "10 - Upstream Status için zone up_mem",
			Path:        filepath.Join(exRoot, "10_upstream_status_zone"),
		},
		{
			ID:          "11_zone_sync_api",
			DefaultName: "11 - Zone Sync ve API ile Durum Sorgulama",
			Path:        filepath.Join(exRoot, "11_zone_sync_api"),
		},
		{
			ID:          "12_logs_and_variables",
			DefaultName: "12 - Loglar ve Değişkenler",
			Path:        filepath.Join(exRoot, "12_logs_and_variables"),
		},
		{
			ID:          "13_include_modular_config",
			DefaultName: "13 - include Directive ve Modüler Config",
			Path:        filepath.Join(exRoot, "13_include_modular_config"),
		},
		{
			ID:          "14_multi_domain_static",
			DefaultName: "14 - Birden Fazla Domain ile Statik Site",
			Path:        filepath.Join(exRoot, "14_multi_domain_static"),
		},
		{
			ID:          "15_https_tls",
			DefaultName: "15 - HTTPS (TLS) Quick Lab",
			Path:        filepath.Join(exRoot, "15_https_tls"),
		},
		{
			ID:          "16_shared_zone_sizing",
			DefaultName: "16 - Shared Memory Zone Boyutlandırma",
			Path:        filepath.Join(exRoot, "16_shared_zone_sizing"),
		},
		{
			ID:          "17_verify_nginx_user",
			DefaultName: "17 - NGINX Kullanıcısını Doğrulama",
			Path:        filepath.Join(exRoot, "17_verify_nginx_user"),
		},
		{
			ID:          "18_osi_layers_usecases",
			DefaultName: "18 - OSI Layer 4/7 & Use-Case Mapping",
			Path:        filepath.Join(exRoot, "18_osi_layers_usecases"),
		},
	}
}
