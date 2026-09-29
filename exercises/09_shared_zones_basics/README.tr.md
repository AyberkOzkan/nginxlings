# 09 – Shared Memory Zones Temelleri

## Amaç

Bu egzersizde ilk **shared memory zone**'unu tanımlayıp rate limiting için
kullanacaksın. Bir zone'un bir kez, bir ad ve boyutla oluşturulduğunu ve sonra
gereken her yerde adıyla çağrıldığını göreceksin.

---

## Arka plan – shared memory zone'lar

NGINX birden fazla worker process çalıştırır. Hepsinin üzerinde anlaşması gereken
durum, örneğin bir istemcinin kaç istek yaptığı, tek bir worker'ın belleğinde
tutulamaz; NGINX bunu **shared memory zone**'larda saklar. Bir zone bir kez ad ve
boyutla tanımlanır, sonra adıyla kullanılır. Rate limiting, bağlantı sınırlama,
upstream durumu ve cache aynı şekilde çalışır.

Rate limiting için:

- `limit_req_zone` (`http` context'inde) zone'u tanımlar. Bir **anahtar** (neye
  göre sayılacağı, örneğin istemci başına), `zone=<ad>:<boyut>` ve `10r/s` ya da
  `30r/m` gibi bir `rate=` alır.
- `limit_req` (`http`, `server` veya `location` içinde) bir zone'u adıyla uygular.
  `burst=`, fazla isteklerin bir kısmının kuyrukta beklemesine izin verir;
  `nodelay` kuyruktaki istekleri aralıklandırmak yerine hemen sunar.
- Reddedilen bir istek, `limit_req_status` başka bir şey demedikçe 503 alır.

Genel bir örnek (bu lab'dakinden farklı ad, anahtar ve değerler):

    limit_req_zone $server_name zone=per_site:1m rate=50r/s;

    server {
        location /search/ {
            limit_req zone=per_site burst=20 nodelay;
        }
    }

`return` rewrite fazında, `limit_req` ise preaccess fazında çalışır; bu yüzden
`return` kullanan bir location hiçbir zaman sınırlanmaz.

Kaynak: [ngx_http_limit_req_module](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html).

---

## Lab ortamı

- `Dockerfile`, `nginx.conf` dosyanı bir `nginx:1.28.3-alpine` image'ına kopyalar;
  `start.sh` NGINX'i foreground'da çalıştırır.
- `nginx.conf` içinde 8080 portunda `/usr/share/nginx/html` altındaki `index.html`'i
  sunan bir `server` hazır; `location /` boş.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i build eder,
`nginx -t` çalıştırır, etkin yapılandırmayı `nginx -T` ile okur (yorum satırları
dikkate alınmaz) ve container'ı başlatır (host portu 19090). NGINX'in hazır
olmasını container'ın **içinden** bekler, böylece bekleme host istemcisinin
hakkını harcamaz; ardından host'tan 20 paralel istek gönderip durum kodlarını
sayar.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `http` bloğunda `req_zone` adında, `64k` boyutunda bir rate-limit zone'u tanımla.
   Anahtarı istemci adresinin ikili (binary) biçimi, `$binary_remote_addr` olmalı.
   Tek bir istemciden gelen 20 eşzamanlı isteğin hepsinin geçemeyeceği kadar düşük
   bir rate seç.
2. `req_zone`'u `location /` içinde uygula.
3. `location /` dosyayı sunmaya devam etsin.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- Etkin yapılandırma `req_zone`'u `64k` boyutu ve `$binary_remote_addr` anahtarıyla
  tanımlamalı ve bir `limit_req` onu kullanmalı.
- 20 paralel isteğin en az biri HTTP 200 almalı, en az biri reddedilmeli (429 veya
  503).

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `No active limit_req_zone yet`: zone tanımlanmamış ya da hâlâ yorum satırı.
- `NGINX rejected the configuration: "limit_req_zone" directive is not allowed
  here`: zone'lar `server` veya `location` içinde değil, `http` bloğunda tanımlanır.
- `must create a zone named req_zone with a size of 64k`: `zone=<ad>:<boyut>`
  parametresine bak.
- `The key of req_zone must be ...`: anahtar, `limit_req_zone`'un ilk parametresidir.
- `Nothing uses req_zone yet`: bir `limit_req` ona başvurmadıkça zone tanımlamak
  hiçbir şeyi sınırlamaz.
- `All 20 parallel requests got through`: rate ya da burst fazla cömert, veya
  `location /` `return` ile cevap veriyor.
- `requests got no answer within 2s instead of being rejected`: fazla istekler
  `burst` ile kuyruğa alınıp yavaşça bırakılıyor. Beklemeli mi, reddedilmeli mi?

---

## Pekiştirme soruları

- İstek sayaçları neden her worker process'te ayrı ayrı değil de shared memory'de
  tutulmalı?
- Reddedilen bir istek varsayılan olarak hangi durum kodunu alır ve bunu nasıl 429
  yaparsın?
- `burst=5`'in `nodelay` ile ve `nodelay` olmadan farkı nedir?
- Aynı amaç için `$binary_remote_addr` neden `$remote_addr`'dan daha iyi bir
  anahtardır?
