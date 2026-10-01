# 09 – Paylaşılan Bellek Bölgelerinin Temelleri

## Amaç

Bu egzersizde ilk **paylaşılan bellek bölgeni (shared memory zone)** tanımlayıp hız
sınırlama (rate limiting) için kullanacaksın. Bir bölgenin bir kez, bir ad ve
boyutla oluşturulduğunu ve sonra gereken her yerde adıyla çağrıldığını göreceksin.

---

## Arka plan – paylaşılan bellek bölgeleri

NGINX birden fazla işçi süreç (worker process) çalıştırır. Hepsinin üzerinde
anlaşması gereken durum, örneğin bir istemcinin kaç istek yaptığı, tek bir işçi
sürecin belleğinde tutulamaz; NGINX bunu **paylaşılan bellek bölgelerinde** saklar.
Bir bölge bir kez ad ve boyutla tanımlanır, sonra adıyla kullanılır. Hız sınırlama,
bağlantı sınırlama, sunucu grubu (upstream) durumu ve önbellek (cache) aynı şekilde
çalışır.

Hız sınırlama için:

- `limit_req_zone` (`http` bağlamında (context)) bölgeyi tanımlar. Bir **anahtar**
  (neye göre sayılacağı, örneğin istemci başına), `zone=<ad>:<boyut>` ve `10r/s` ya
  da `30r/m` gibi bir `rate=` alır.
- `limit_req` (`http`, `server` veya `location` içinde) bir bölgeyi adıyla uygular.
  `burst=`, fazla isteklerin bir kısmının kuyrukta beklemesine izin verir;
  `nodelay` kuyruktaki istekleri aralıklandırmak yerine hemen sunar.
- Reddedilen bir istek, `limit_req_status` başka bir şey demedikçe 503 alır.

Genel bir örnek (bu laboratuvardakinden farklı ad, anahtar ve değerler):

    limit_req_zone $server_name zone=per_site:1m rate=50r/s;

    server {
        location /search/ {
            limit_req zone=per_site burst=20 nodelay;
        }
    }

`return` rewrite fazında, `limit_req` ise preaccess fazında çalışır; bu yüzden
`return` kullanan bir `location` hiçbir zaman sınırlanmaz.

Kaynak: [ngx_http_limit_req_module](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html).

---

## Laboratuvar ortamı

- `Dockerfile`, `nginx.conf` dosyanı bir `nginx:1.28.3-alpine` imajına (image)
  kopyalar; `start.sh` NGINX'i ön planda (foreground) çalıştırır.
- `nginx.conf` içinde 8080 portunda `/usr/share/nginx/html` altındaki `index.html`'i
  sunan bir `server` hazır; `location /` boş.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) imajı oluşturur (build),
`nginx -t` çalıştırır, etkin yapılandırmayı `nginx -T` ile okur (yorum satırları
dikkate alınmaz) ve konteyneri (container) başlatır (ana makine (host) portu 19090).
NGINX'in hazır olmasını konteynerin **içinden** bekler, böylece bekleme ana makine
istemcisinin hakkını harcamaz; ardından ana makineden 20 paralel istek gönderip
durum kodlarını sayar.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `http` bloğunda `req_zone` adında, `64k` boyutunda bir hız sınırlama bölgesi
   tanımla. Anahtarı istemci adresinin ikili (binary) biçimi, `$binary_remote_addr`
   olmalı. Tek bir istemciden gelen 20 eşzamanlı isteğin hepsinin geçemeyeceği kadar
   düşük bir hız seç.
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

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `No active limit_req_zone yet`: bölge tanımlanmamış ya da hâlâ yorum satırı.
- `NGINX rejected the configuration: "limit_req_zone" directive is not allowed
  here`: bölgeler `server` veya `location` içinde değil, `http` bloğunda tanımlanır.
- `must create a zone named req_zone with a size of 64k`: `zone=<ad>:<boyut>`
  parametresine bak.
- `The key of req_zone must be ...`: anahtar, `limit_req_zone`'un ilk parametresidir.
- `Nothing uses req_zone yet`: bir `limit_req` ona başvurmadıkça bölge tanımlamak
  hiçbir şeyi sınırlamaz.
- `All 20 parallel requests got through`: hız ya da `burst` fazla cömert, veya
  `location /` `return` ile yanıt veriyor.
- `requests got no answer within 2s instead of being rejected`: fazla istekler
  `burst` ile kuyruğa alınıp yavaşça bırakılıyor. Beklemeli mi, reddedilmeli mi?

---

## Pekiştirme soruları

- İstek sayaçları neden her işçi süreçte ayrı ayrı değil de paylaşılan bellekte
  tutulmalı?
- Reddedilen bir istek varsayılan olarak hangi durum kodunu alır ve bunu nasıl 429
  yaparsın?
- `burst=5`'in `nodelay` ile ve `nodelay` olmadan farkı nedir?
- Aynı amaç için `$binary_remote_addr` neden `$remote_addr`'dan daha iyi bir
  anahtardır?
