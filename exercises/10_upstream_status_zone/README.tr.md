# 10 – Upstream Shared Memory (zone up_mem)

## Amaç

Bu egzersizde HTTP upstream durumunun shared memory’de tutulmasını sağlayan
upstream `zone` kullanımını pratik edeceksin:

    zone up_mem 64k;

Yapacakların:

- İki backend içeren bir `upstream` tanımlamak:

      127.0.0.1:8081
      127.0.0.1:8082

- Upstream state bilgisini shared memory’de tutmak için:

      zone up_mem 64k;

  satırını kullanmak.

- İstekleri bu upstream’e `proxy_pass` ile yönlendirmek.
- Hangi backend’in yanıt verdiğini response header üzerinden göstermek:

      X-Upstream: <ip:port>

Doğrulama sırasında backend’lerden biri bilerek durdurulur. Bu durumdan sonra
isteklerin “down” olan backend’e gitmemesi beklenir.

---

## Ne değiştireceksin?

Düzenleyeceğin dosya:

    nginx.conf


Doğrulama script’i aynı Docker network içinde iki basit HTTP backend’i ayrı container olarak başlatır:
- Backend A: 127.0.0.1:8081
- Backend B: 127.0.0.1:8082

Senin görevin, NGINX’i bu iki backend arasında load balance edecek şekilde
yapılandırmak ve hangi backend’in seçildiğini görünür hale getirmektir.

---

## Gereksinimler

`nginx.conf` içinde şunlar olmalı:

1) Shared memory zone kullanan upstream:

      upstream backend {
          zone up_mem 64k;
          server 127.0.0.1:8081;
          server 127.0.0.1:8082;
      }

2) Upstream’e proxy yapan bir location:

      location / {
          proxy_pass http://backend;
      }

3) Seçilen upstream’i gösteren bir header:

      add_header X-Upstream $upstream_addr always;

---

## Başarı kriterleri

Otomatik doğrulama başarılı sayılması için:

- Image build olmalı ve `nginx -t` hatasız geçmeli.
- İlk aşamada iki backend’e de istek gittiği görülmeli (dağıtım gözlenmeli).
- 8082 backend’i durdurulduktan sonra:

    - İstekler 8081 üzerinden cevaplanmaya devam etmeli.
    - Header bazen retry zinciri şeklinde görünebilir:

        X-Upstream: <addr>:8082, <addr>:8081

    Bu normaldir; önce 8082 denenip sonra 8081’e retry edildiğini gösterir.
    - Sadece 8082 görünen (8081 içermeyen) bir upstream değeri görülmemeli.


---

## İpuçları

- `zone` direktifi mutlaka `upstream` bloğunun içinde olmalı.
- Header’ı, proxy yaptığın `location` içinde eklemek en doğru yaklaşımdır.
- `max_fails` / `fail_timeout` ayarları failover davranışını daha belirgin yapabilir.
