# 11 – Zone Synchronization & API (zone_sync)

## Amaç

Zone Sync durumunu temsil eden bir JSON API endpoint’i yayınlamak.

Bu egzersizde NGINX’i şu path için JSON döndürecek şekilde yapılandıracaksın:

    /api/8/stream/zone_sync

Response şu özelliklere sahip olmalı:

- HTTP 200
- `Content-Type: application/json`
- Geçerli JSON (çıktı `jq` ile parse edilebilmeli)
- JSON içinde en az şu alanlar bulunmalı:

    .zones.sync_zone.state
    .zones.sync_zone.last_sync
    .zones.sync_zone.peers

`peers` alanı tam olarak 2 elemanlı bir dizi olmalı.

---

## Görev

Bu egzersizin `nginx.conf` dosyasını düzenle ve NGINX’i şu isteğe JSON döndürecek
hale getir:

    /api/8/stream/zone_sync

Gereksinimler:

1. Path tam eşleşmeli (prefix match değil).
2. Response JSON olarak dönmeli (`application/json`).
3. Response body geçerli JSON olmalı ve şu alanları içermeli:
   - `zones.sync_zone.state`
   - `zones.sync_zone.last_sync`
   - `zones.sync_zone.peers` (2 eleman)

---

## Başarı kriterleri

Bu egzersiz doğrulaması başarılı olduğunda ve şu koşullar sağlandığında tamamlanmış sayılır:

- API path HTTP 200 dönüyor.
- Response `Content-Type: application/json` içeriyor.
- JSON geçerli ve gerekli alanları içeriyor.
- `peers` alanında tam olarak 2 peer var.

---

## İpuçları

- Exact-match location kullan:

    location = /api/8/stream/zone_sync { ... }

- JSON’u statik olarak döndürebilirsin (dosya üzerinden ya da direkt return ile).
  Önemli olan Content-Type ve JSON formatı.
- Fail olursa, ekranda gördüğün hata çıktısına göre konfigürasyonu düzeltip tekrar dene.
