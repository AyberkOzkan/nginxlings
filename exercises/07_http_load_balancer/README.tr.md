# 07 – HTTP Load Balancer (Layer 7)

## Amaç

Bu egzersizde NGINX’i **HTTP load balancer** olarak kullanacaksın.

Hedeflerin:

- Aynı container içinde iki adet backend HTTP sunucu çalışsın:
  - 127.0.0.1:8081
  - 127.0.0.1:8082
- NGINX, `upstream backend` bloğu ile bu iki backend arasında **round-robin** load balancing yapsın.
- Frontend olarak 8080 portunda bir `server` bloğu olsun.
- `/` istekleri `proxy_pass http://backend;` ile upstream’e gitsin.
- Loglarda `$upstream_addr` üzerinden isteklerin iki backend’e dağıldığını görebilesin.

Tüm bunlar, senin yazdığın `nginx.conf` kullanılarak **Docker container** içinde çalışacak.

---

## Arka plan – upstream ve round-robin

NGINX’te basit bir HTTP load balancer yapısı genelde şöyle görünür:

    upstream backend {
        server 127.0.0.1:8081;
        server 127.0.0.1:8082;
    }

    server {
        listen 8080;
        server_name localhost;

        location / {
            proxy_pass http://backend;
        }
    }

Varsayılan davranış **round-robin**’dir: Her yeni istek upstream listesindeki bir sonraki backend’e gider.

Log formatına `$upstream_addr` eklersen, her isteğin hangi backend’e gittiğini görebilirsin:

    log_format lb_main
        '$remote_addr - $remote_user [$time_local] '
        '"$request" $status $body_bytes_sent '
        '"$http_referer" "$http_user_agent" '
        'upstream=$upstream_addr';

    access_log /dev/stdout lb_main;

---

## Bu egzersizde ne yapıyorsun?

Bu dizinde çalışacaksın:

    07_http_load_balancer/

Dosyalar:

- `nginx.conf`  
  NGINX yapılandırma iskeleti. `upstream backend` ve `server` bloğu için TODO yorumları var.
- `Dockerfile`  
  `nginx:alpine` tabanlı bir image oluşturur, `nginx.conf` ve `start.sh` dosyalarını image içine kopyalar.
- `start.sh`  
  Container içinde iki adet backend HTTP sunucuyu başlatır (127.0.0.1:8081 ve 127.0.0.1:8082) ve NGINX’i foreground’da çalıştırır.
- `check.sh`  
  Bu egzersizi otomatik olarak test eder:
  - Image’i build eder,
  - Container’ı başlatır,
  - Birkaç HTTP isteği gönderir,
  - Loglardan isteklerin her iki backend’e gittiğini kontrol eder.

Senin asıl düzenleyeceğin dosya: `nginx.conf`.

---

## Adımlar

1. Dizine geç

    cd exercises/07_http_load_balancer

2. `nginx.conf` dosyasını incele

3. `upstream backend` bloğunu tamamla

   Aşağıdaki gibi iki backend eklemelisin:

    upstream backend {
        server 127.0.0.1:8081;
        server 127.0.0.1:8082;
    }

4. `location /` içinde proxy_pass ayarla

   Frontend 8080 portunda dinleyen server, bütün `/` isteklerini upstream’e yönlendirmeli:

    server {
        listen 8080;
        server_name localhost;

        location / {
            proxy_pass http://backend;
        }
    }

   İstersen buraya ekstra `proxy_set_header` direktifleri de ekleyebilirsin (zorunlu değil).

5. Otomatik testi çalıştır

    ./check.sh

   Script şunları yapar:

   - `Dockerfile` kullanarak image’i build eder (örn. `nginxlings_http_lb_07`),
   - Bu image ile bir container başlatır,
   - Container içinde:
     - 127.0.0.1:8081 ve 127.0.0.1:8082 üzerinde iki backend HTTP sunucu açılır,
     - NGINX 8080 portunda istek bekler,
   - Host tarafından birkaç kez `http://127.0.0.1:18080/` adresine istek gönderir,
   - Container log’larında `$upstream_addr` üzerinden isteklerin hem 127.0.0.1:8081’e hem de 127.0.0.1:8082’ye gittiğini kontrol eder.

Hata varsa `check.sh` çıktısında görebilirsin; loglardan hangi backend’lere kaç istek gittiği gösterilir.

---

## Başarı kriterleri

Bu egzersiz başarılı sayılması için:

- `docker` komutu sistemde çalışır durumda olmalı.
- `Dockerfile`, `start.sh` ve `nginx.conf` mevcut olmalı ve image başarıyla build edilmeli.
- Container içinde `nginx -t` komutu yapılandırmayı geçerli bulmalı.
- `upstream backend` bloğunda **en az iki server** olmalı:
  - 127.0.0.1:8081
  - 127.0.0.1:8082
- `server` bloğu 8080 portunu dinlemeli ve:

    location / {
        proxy_pass http://backend;
    }

  gibi bir yapı içermeli (yani `/` istekleri upstream’e gitmeli).
- Log formatı `$upstream_addr` içermeli ve `access_log` bu formatı kullanmalı.
- `check.sh` çıktılarına göre:
  - En az bir istek 127.0.0.1:8081’e gitmiş olmalı,
  - En az bir istek 127.0.0.1:8082’ye gitmiş olmalı.

---

## İpuçları

- `nginx.conf` içinde küçük bir syntax hatası bile container’ın başlamasına engel olur; `check.sh` bu tür hatalarda `nginx -t` çıktısını gösterecektir.
- `$upstream_addr` loglara eklenmediyse, backend dağılımını görmek çok zorlaşır; bu nedenle `log_format lb_main` tanımını silme.
- `check.sh` başarısız olursa, çıktıdaki log örneklerini dikkatle incele; genelde hangi backend’lerin kullanıldığı doğrudan satırlarda görünecektir.

