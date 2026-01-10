# 09 – Shared Memory Zones Temelleri

## Amaç

Bu egzersizde NGINX’te shared memory zone kavramını rate limiting üzerinden pratik edeceksin.

Yapacakların:

- `limit_req_zone` ile shared memory zone tanımlamak,
- `limit_req` ile bu zone’u kullanarak rate limit uygulamak,
- Zone boyutunun konfigürasyondan geldiğini göstermek (örneğin `:64k`).

Otomatik kontrol şunları yapar:

- Konfigürasyonun syntax kontrolünü yapar,
- `limit_req_zone` içinde `req_zone:64k` tanımını arar,
- `limit_req zone=req_zone` kullanımını arar,
- Hızlı istekler gönderip rate limiting’in gerçekten çalıştığını doğrular.

---

## Ne değiştireceksin?

Düzenleyeceğin dosya:

    nginx.conf


---

## Gereksinimler

`nginx.conf` içinde şunlar bulunmalı:

1) Shared memory zone tanımı:

    limit_req_zone $binary_remote_addr zone=req_zone:64k rate=1r/m;

2) Bu zone’un kullanıldığı yer:

    limit_req zone=req_zone burst=1 nodelay;

Notlar:

- Zone adı `req_zone` olmalı.
- Zone boyutu `64k` olmalı (kontrol ediliyor).

---

## Başarı kriterleri

Egzersiz başarılı sayılması için:

- `nginx -t` hatasız geçmeli,
- Konfigürasyonda `zone=req_zone:64k` geçmeli,
- Konfigürasyonda `limit_req zone=req_zone` geçmeli,
- Hızlı istek yağmurunda:
  - En az bir istek 200 dönmeli,
  - En az bir istek rate-limit’e takılmalı (HTTP 429 veya 503).

---

## İpuçları

- `limit_req_zone` genelde `http` context’inde tanımlanır.
- `limit_req` genelde `server` veya `location` içinde uygulanır.
- Rate limit hiç tetiklenmiyorsa `rate` ve `burst` değerlerini küçült.
