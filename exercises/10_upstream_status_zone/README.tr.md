# 10 – Upstream Shared Memory (zone up_mem)

## Amaç

Bu egzersizde HTTP upstream durumunun shared memory’de tutulmasını sağlayan
upstream `zone` kullanımını pratik edeceksin:

    zone up_mem 64k;

Yapacakların:

- İki backend içeren bir `upstream` tanımlamak:

      ex10_backend1:8081
      ex10_backend2:8082

- Upstream state bilgisini shared memory’de tutmak için:

      zone up_mem 64k;

  satırını kullanmak.

- İstekleri bu upstream’e `proxy_pass` ile yönlendirmek.
- Hangi backend’in yanıt verdiğini response header üzerinden göstermek:

      X-Upstream: <ip:port>

Doğrulama sırasında backend’lerden birinin HTTP sunucusu bilerek durdurulur;
container çalışmaya devam ettiği için ona giden bağlantılar reddedilir. Bu
durumdan sonra her isteğin diğer backend üzerinden başarıyla cevaplanması beklenir.

---

## Ne değiştireceksin?

Düzenleyeceğin dosya:

    nginx.conf

Doğrulama script’i NGINX’i ve iki basit HTTP backend’i aynı Docker network içinde
ayrı container’lar olarak başlatır:
- Backend A: `ex10_backend1:8081`
- Backend B: `ex10_backend2:8082`

Senin görevin, NGINX’i bu iki backend arasında load balance edecek şekilde
yapılandırmak ve hangi backend’in seçildiğini görünür hale getirmektir.
Bu endpoint’ler için `127.0.0.1` kullanma; NGINX container’ının içinde bu adres
backend’leri değil, NGINX container’ının kendisini gösterir.

Doğrulayıcı, backend container’ları oluşturulmadan önce config’i kontrol eder.
`http` context’inde Docker’ın `127.0.0.11` adresindeki DNS resolver’ını kullan ve
her upstream server için `resolve` parametresiyle runtime name resolution’ı aç.

---

## Gereksinimler

`nginx.conf` içinde şunlar olmalı:

1. `http` context’i içinde bir Docker DNS `resolver` tanımı.
2. `ex10_backend1:8081` ve `ex10_backend2:8082` adreslerini içeren `backend`
   isimli bir upstream; iki server tanımı da runtime `resolve` kullanmalı.
3. Bu upstream içinde `zone up_mem 64k;` direktifi.
4. İstekleri `backend` upstream’ine proxy eden bir location.
5. Değeri `$upstream_addr` olan bir `X-Upstream` response header’ı.

---

## Başarı kriterleri

Otomatik doğrulama başarılı sayılması için:

- Image build olmalı ve `nginx -t` hatasız geçmeli.
- İlk aşamada iki backend’e de istek gittiği görülmeli (dağıtım gözlenmeli).
- 8082 backend’i durdurulduktan sonra:

    - Sonraki 12 isteğin tamamı 8081 üzerinden başarıyla (HTTP 200) cevaplanmalı.
    - Header bazen retry zinciri şeklinde görünebilir:

        X-Upstream: <addr>:8082, <addr>:8081

    Bu normaldir; önce 8082 denenip sonra 8081’e retry edildiğini gösterir.
    - Sadece 8082 görünen (8081 içermeyen) bir upstream değeri görülmemeli.

---

## İpuçları

- `zone` direktifi mutlaka `upstream` bloğunun içinde olmalı.
- Docker resolver’ı `upstream` bloğunda değil, `http` context’inde olmalı.
- Backend container’ları başladıktan sonra isimlerin çözülebilmesi için iki
  upstream `server` tanımına da `resolve` parametresini ekle.
- Header’ı, proxy yaptığın `location` içinde eklemek en doğru yaklaşımdır.
- `max_fails` / `fail_timeout` ayarları failover davranışını daha belirgin yapabilir.
