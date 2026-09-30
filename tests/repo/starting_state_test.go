package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nginxlings/tests/testutil"
)

func TestRepoExercisesStartUnsolved(t *testing.T) {
	root := testutil.RepoRoot(t)

	tests := []struct {
		name      string
		path      string
		required  []string
		forbidden []string
	}{
		{
			name: "static server config",
			path: filepath.Join(root, "exercises", "01_static_server", "nginx.conf"),
			required: []string{
				"TODO: point the document root",
				"root missing;",
				"index missing.html;",
			},
			forbidden: []string{"root .;", "index index.html;"},
		},
		{
			name:      "static server content",
			path:      filepath.Join(root, "exercises", "01_static_server", "index.html"),
			required:  []string{"TODO: replace this placeholder"},
			forbidden: []string{"Merhaba Nginxlings"},
		},
		{
			name: "shared zone sizing config",
			path: filepath.Join(root, "exercises", "16_shared_zone_sizing", "nginx.conf"),
			required: []string{
				"zone=req_zone:64k rate=1r/m;",
				"try_files /index.html =404;",
			},
			// burst=0 is rejected by nginx -t, and return would bypass limit_req.
			forbidden: []string{"burst=0", "return 200"},
		},
		{
			name: "worker user config",
			path: filepath.Join(root, "exercises", "04_worker_user", "nginx.conf"),
			required: []string{
				"TODO: the workers must be able to read the site",
				"user www-data;",
			},
			forbidden: []string{"user webapp", "user nginx webapp", "user root"},
		},
		{
			name:      "install and verify Dockerfile",
			path:      filepath.Join(root, "exercises", "05_install_verify", "Dockerfile"),
			required:  []string{"TODO: install NGINX"},
			forbidden: []string{"apt-get install"},
		},
		{
			name: "HTTP load balancer config",
			path: filepath.Join(root, "exercises", "07_http_load_balancer", "nginx.conf"),
			required: []string{
				"TODO: replace this placeholder with both local backend servers",
				"TODO: proxy requests to the backend upstream",
			},
			forbidden: []string{
				"server 127.0.0.1:8081;",
				"server 127.0.0.1:8082;",
				"proxy_pass http://backend;",
			},
		},
		{
			name: "upstream status zone config",
			path: filepath.Join(root, "exercises", "10_upstream_status_zone", "nginx.conf"),
			required: []string{
				"TODO: enable the required shared memory zone",
				"TODO: replace this placeholder with both Docker backend servers",
				"TODO: expose the selected upstream",
				"TODO: proxy requests to the backend upstream",
			},
			forbidden: []string{
				"zone up_mem 64k;",
				"server ex10_backend1:8081",
				"server ex10_backend2:8082",
				"add_header X-Upstream $upstream_addr",
				"proxy_pass http://backend;",
			},
		},
		{
			name: "logs and variables config",
			path: filepath.Join(root, "exercises", "12_logs_and_variables", "nginx.conf"),
			required: []string{
				`TODO: define a log format named "main"`,
				`TODO: write the access log in the "main" format to /dev/stdout`,
			},
			forbidden: []string{"log_format main '", "access_log /dev/stdout main;"},
		},
		{
			name:      "NGINX worker user config",
			path:      filepath.Join(root, "exercises", "17_verify_nginx_user", "nginx.conf"),
			required:  []string{"TODO: configure NGINX workers to run as the nginx user"},
			forbidden: []string{"user nginx;"},
		},
		{
			name: "OSI layer proxy config",
			path: filepath.Join(root, "exercises", "18_osi_layers_usecases", "nginx.conf"),
			required: []string{
				"TODO: load the NGINX stream module",
				"TODO: define an HTTP upstream and proxy this location to it",
				"TODO: expose the selected L7 upstream",
				"TODO: add a stream context that proxies port 9000",
			},
			forbidden: []string{
				"load_module /usr/lib/nginx/modules/ngx_stream_module.so;",
				"upstream http_pool",
				"proxy_pass http://http_pool;",
				"add_header X-L7-Upstream",
				"stream {",
				"upstream tcp_pool",
				"proxy_pass tcp_pool;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("%s: %v", tt.path, err)
			}
			content := string(data)
			for _, required := range tt.required {
				testutil.AssertContains(t, tt.path, content, required)
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(content, forbidden) {
					t.Fatalf("%s unexpectedly contains completed solution %q", tt.path, forbidden)
				}
			}
		})
	}

	legacyConfig := filepath.Join(root, "exercises", "07_http_load_balancer", "nginx.conf.default")
	if _, err := os.Stat(legacyConfig); err == nil {
		t.Fatalf("unused duplicate config still exists: %s", legacyConfig)
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect %s: %v", legacyConfig, err)
	}
}
