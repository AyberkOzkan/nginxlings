# 13 – Include & Modular Config Design

## Amaç

Bu egzersizde `include` directive’inin **gerçek davranışını** öğreneceksin.

Hedef:
- `nginx.conf` dosyasını sade tutup, yapılandırmayı parçalara bölmek
- `include` edilen dosyaların içeriğinin, `include` satırına **birebir eklenmiş gibi** işlendiğini göstermek
- Context kurallarını (http/server/location) pratikte görmek

---

## Konu

- `include` directive
- Modüler config tasarımı
- Context davranışı: `include` nereye yazıldıysa içerik **orada** parse edilir

---

## Görev

Bu egzersizde yapı şu şekilde tasarlandı:

- `nginx.conf` içinde `http { ... }` bloğu var
- `http` içinde `conf.d/*.conf` include edilerek **server** dosyaları yüklenecek
- `conf.d/10-server.conf` içinde bir `server { ... }` var
- `server` içinde de `server.d/*.conf` include edilerek **location** parçaları yüklenecek

Senin yapman gerekenler:

1. `nginx.conf` dosyasında, `http {}` içine aşağıdaki include’ı ekle:
   - /etc/nginx/conf.d/*.conf

2. `conf.d/10-server.conf` içinde, `server {}` içine aşağıdaki include’ı ekle:
   - /etc/nginx/server.d/*.conf

3. `server.d/20-modular.conf` içinde `/modular` endpoint’ini oluştur:
   - `GET /modular` cevabı: `include-ok\n`

4. `server.d/30-modular-extra.conf` içinde `/modular2` endpoint’ini oluştur:
   - `GET /modular2` cevabı: `include2-ok\n`

---

## Beklenen sonuçlar

Bu endpoint’ler çalışmalı:

- `GET /`        → `base-ok\n`
- `GET /modular` → `include-ok\n`
- `GET /modular2`→ `include2-ok\n`

---

## Başarı kriterleri

- Konfigürasyon geçerli olmalı (`nginx -t` hata vermemeli)
- `/`, `/modular`, `/modular2` endpoint’leri doğru çıktıyı dönmeli
- `include /etc/nginx/conf.d/*.conf;` satırı `http {}` içinde olmalı
- `include /etc/nginx/server.d/*.conf;` satırı `server {}` içinde olmalı
- `location` blokları `server` context dışında olmamalı (yanlış include yeri hata üretir)

---

## İpucu

`include` bir “kısayol” değildir; içerik gerçekten o noktaya eklenmiş gibi parse edilir.

Bu yüzden:
- `location {}` ancak `server {}` içinde anlamlıdır
- `server {}` ancak `http {}` içinde anlamlıdır
- Yanlış yere include koyarsan NGINX “directive is not allowed here” benzeri hata verir
