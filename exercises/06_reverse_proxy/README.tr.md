# 06 – HTTP Reverse Proxy Temelleri

## Amaç

Bu egzersizde NGINX’i basit bir **HTTP reverse proxy** olarak
yapılandıracaksın.

Hedeflerin:

- `http://127.0.0.1:8080` adresinde dinleyen küçük bir backend uygulamasına sahip olmak,
- NGINX’i `127.0.0.1:8081` üzerinde dinleyecek şekilde ayarlamak,
- `proxy_pass` kullanarak gelen istekleri backend’e yönlendirmek,
- İstemcinin sadece NGINX ile konuşmasını, backend’i doğrudan görmemesini sağlamak.

Bu dizindeki otomatik kontrol script’i, yapılandırmanı doğrulayacak.

---

## Arkaplan – Reverse proxy nedir?

Reverse proxy, istemciden gelen istekleri:

- Kendisi karşılayan (örneğin `127.0.0.1:8081`),
- Arkadaki bir veya birden fazla backend sunucuya ileten,
- Backend’den gelen yanıtı tekrar istemciye dönen bir katmandır.

NGINX için minimal bir reverse proxy örneği:

    http {
        server {
            listen 8081;
            server_name localhost;

            location / {
                proxy_pass http://127.0.0.1:8080;
            }
        }
    }

Bu örnekte:

- **İstemci**, NGINX’e `127.0.0.1:8081` üzerinden bağlanır,
- NGINX isteği `127.0.0.1:8080` adresindeki backend’e iletir,
- Backend yanıt döner,
- NGINX bu yanıtı istemciye iletir.

İstemci backend’i doğrudan görmez, sadece NGINX’i görür.

---

## Bu egzersizde ne yapıyorsun?

Şu dizinde çalışıyorsun:

    06_reverse_proxy/

Önemli dosyalar:

- `nginx.conf` – NGINX yapılandırman,
- `backend_app.py` – basit bir HTTP backend (örnek bir yanıt döner),
- `check.sh` – otomatik kontrol script’i.

Görevin:

1. `nginx.conf` dosyasını şu şekilde ayarlamak:

   - NGINX `127.0.0.1:8081` üzerinde dinlemeli (HTTP),
   - `http` bloğu altında bir `server` bloğu olmalı,
   - `server` içinde bir `location /` bloğu olmalı ve:
     - `proxy_pass` ile istekleri `http://127.0.0.1:8080` adresine yönlendirmeli.

2. Diğer egzersizlerle uyumlu temel iskeleti korumak:

   - Log’lar için `logs/` dizinini kullanmak (`error_log`, `access_log`),
   - `events { ... }` ve `http { ... }` context’lerini geçerli tutmak.

3. Backend uygulaması, kontrol script’i tarafından otomatik başlatılacaktır.
   Bu dosyayı **elle çalıştırman gerekmez**.

Bu egzersiz için `backend_app.py` veya `check.sh` dosyalarına dokunman gerekmez.

---

## Başarı kriterleri

Egzersiz **başarılı** sayılması için:

- `nginx.conf` syntax olarak geçerli olmalı ve `nginx -t` testini,
  bu dizini `-p` prefix olarak kullanarak geçmeli,
- NGINX yapılandırmanla sorunsuz şekilde başlayabilmeli (runtime hatası olmamalı),
- `http://127.0.0.1:8081/` adresine gelen istekler, arkadaki
  `http://127.0.0.1:8080/` backend’ine yönlenmiş olmalı,
- İstemciye dönen içerik, backend’in ürettiği yanıt olmalı
  (otomatik kontrolller bunu doğrular),
- Bu dizindeki otomatik kontrol script’i başarı mesajı vermeli.

---

## İpuçları

- Context hiyerarşisini doğru tut:

  - `events` ve `http` en üst (main) seviyede,
  - `server` → `http` bloğunun içinde,
  - `location` → `server` bloğunun içinde.

- Genellikle çalışan basit bir şablon:

      user  www-data;
      worker_processes auto;

      error_log  logs/error.log;
      pid        logs/nginx.pid;

      events {
          worker_connections 1024;
      }

      http {
          access_log  logs/access.log;

          server {
              listen 8081;
              server_name localhost;

              location / {
                  proxy_pass http://127.0.0.1:8080;
              }
          }
      }

- Gerçek dünyada yaygın olan bazı header ayarlarını da ekleyebilirsin:

      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

  Bu ayarlar temel çalışmayı zorunlu kılmaz ama reverse proxy
  için iyi bir alışkanlıktır.






