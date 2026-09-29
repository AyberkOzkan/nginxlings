# 11 – Zone Synchronization & API (zone_sync)

## Amaç

Küçük bir JSON API gibi davranan bir Zone Sync durum endpoint'i aç: bir JSON
belgesiyle cevap veren, doğru content type ile sunulan tek ve tam bir URL.

---

## Arka plan – tam eşleşmeler, dosyalar ve content type

**Location seçimi.** NGINX her istek için önce tam eşleşme arar
(`location = /yol`); biri eşleşirse hemen o kullanılır. Yoksa en uzun eşleşen
prefix location'ı aklında tutar, sonra regular expression location'larını (`~`,
`~*`) sırayla dener; eşleşen bir regex prefix'e karşı kazanır. `^~` ile işaretli
bir prefix location regex aramasını durdurur.

**URL'yi bir dosyaya eşlemek.**

- `root <dizin>`, istek yolunun tamamını `<dizin>`'in sonuna ekler.
- `alias <yol>`, yolun location'ın eşleştiği kısmını `<yol>` ile değiştirir.

**Content type.** NGINX statik dosyalar için diskteki dosyanın değil, **istek
yolunun** uzantısını `types` haritasında arar (harita genellikle
`include mime.types;` ile yüklenir). Yolun uzantısı yoksa ya da haritada eşleşme
yoksa `default_type` kullanılır; sen ayarlamadıkça bu `text/plain`'dir.

Genel bir örnek (bu lab'dakinden farklı yollar):

    location = /health {
        default_type text/plain;
        return 200 "ok\n";
    }

    location /downloads/ {
        # /downloads/app.zip, /srv/files/app.zip dosyasından okunur
        alias /srv/files/;
    }

Kaynak: [location](https://nginx.org/en/docs/http/ngx_http_core_module.html#location),
[alias](https://nginx.org/en/docs/http/ngx_http_core_module.html#alias),
[default_type](https://nginx.org/en/docs/http/ngx_http_core_module.html#default_type).

---

## Lab ortamı

- `Dockerfile`, `nginx.conf` dosyanı ve verilen `api/zone_sync.json`'u bir
  `nginx:1.28.3-alpine` image'ına kopyalar. JSON dosyası
  `/usr/share/nginx/html/api/zone_sync.json` yoluna yerleşir. Image'da check için
  `curl` ve `jq` de var.
- `nginx.conf` içinde 8080 portunda bir `server` var; `location /` 404 döner.
  `mime.types` include edilmemiş.
- JSON dosyasını değiştirme; check'in istediği alanlar zaten içinde.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i build eder,
`nginx -t` çalıştırır, container'ı başlatır (host portu 18110), container içinden
`/api/8/stream/zone_sync` ister ve durum kodunu, `Content-Type` başlığını ve JSON
alanlarını `jq` ile kontrol eder.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `/api/8/stream/zone_sync` yolunu prefix olarak değil, tam olarak eşleştiren bir
   location ekle.
2. Bu yol için verilen JSON dosyasını sun.
3. Cevabın `Content-Type`'ı `application/json` olsun.

---

## Başarı kriterleri

- `/api/8/stream/zone_sync` HTTP 200 dönmeli.
- Cevapta `Content-Type: application/json` olmalı.
- Gövde geçerli bir JSON olmalı ve `.zones.sync_zone.state`,
  `.zones.sync_zone.last_sync` ile tam iki öğe içeren `.zones.sync_zone.peers`
  alanlarını içermeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `Expected HTTP 200, got 404`: önce hangi location'ın cevap verdiğini sor.
  `location /` ise senin location'ın bu yolla eşleşmiyor. Seninki ise yol JSON
  dosyasına ulaşmıyor: `root` istek yoluyla ne yapar, `alias` ne yapar?
- `Expected Content-Type to include application/json, got text/plain`: dosya
  sunulmuş ama istek yolunun uzantısı olmadığı için NGINX bir tip bulamamış. Yedek
  tipi hangi direktif belirler?
- `Missing JSON field` veya `Expected exactly 2 peers`: gövde verilen dosya değil.
  `zone_sync.json`'u mu sunuyorsun?

---

## Pekiştirme soruları

- `/api/8/stream/zone_sync/` (sonunda eğik çizgiyle) isteğine hangi location cevap
  verir ve neden?
- Aynı server'da `location /api/ { return 403; }` ve
  `location ~ ^/api/8/ { return 410; }` da olsaydı, senin location'ın düz bir
  prefix olduğunda `/api/8/stream/zone_sync`'e hangisi cevap verirdi; `=` ya da
  `^~` ile ne değişir?
- Location'ın `root /usr/share/nginx/html;` kullansaydı NGINX hangi dosyayı açmaya
  çalışırdı?
- NGINX statik bir dosyanın `Content-Type`'ını normalde nasıl seçer? Bu URL için
  `include mime.types;` eklemek neden content type'ı düzeltmez?
