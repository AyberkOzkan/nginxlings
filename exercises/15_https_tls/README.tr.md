# 15 – HTTPS (TLS) Quick Lab

## Amaç

Bu egzersizde NGINX'e **TLS terminasyonu** yaptıracaksın: image içinde hazır
bulunan bir sertifikayla HTTPS üzerinden cevap verecek ve düz HTTP istemcilerini
HTTPS adresine yönlendirecek.

Pratik edeceklerin:

- dinlenen bir soket üzerinde TLS'i açmak,
- NGINX'e bir sertifikayı ve onun private key'ini göstermek,
- host adını ve path'i kaybetmeden HTTP'yi HTTPS'e yönlendirmek.

---

## Arka plan – TLS terminasyonu ve HTTP yönlendirmesi

TLS terminasyonunda NGINX istemciyle TLS handshake'ini yapar, trafiği çözer ve
içindeki düz HTTP isteğini işler. Tipik bir site iki `server` bloğu kullanır:
biri şifreli bağlantıları kabul eder, düz HTTP portundaki diğeri ise istemcilere
yalnızca HTTPS üzerinden tekrar gelmelerini söyler.

Genel bir örnek (bu lab'dakinden farklı ad, port ve dosyalar):

    server {
        listen 8080;
        server_name shop.example;
        return 308 https://$host:8443$request_uri;
    }

    server {
        listen 8443 ssl;
        server_name shop.example;

        ssl_certificate     /etc/ssl/shop/fullchain.pem;
        ssl_certificate_key /etc/ssl/shop/privkey.pem;

        location / {
            root /srv/shop;
        }
    }

Bilinmesi gerekenler:

- TLS, `listen` direktifinin `ssl` parametresiyle açılır. Ayrı `ssl on;`
  direktifi NGINX 1.25.1'de kaldırıldı.
- `ssl_certificate` sertifikayı (PEM; istenirse arkasından zinciri),
  `ssl_certificate_key` ise private key'i gösterir. İkisi de zorunludur ve
  `nginx -t` dosyaları yükler; bu yüzden yanlış bir path ilk istekte değil, testte
  hata verir.
- `server` bloğundaki `return code URL;`, herhangi bir `location` seçilmeden önce
  cevap verir. `$host` istemcinin istediği host adını, `$request_uri` ise orijinal
  path'i ve query string'i korur.
- `/` ile başlayan bir yönlendirme hedefi, mevcut isteğin şeması ve host'u ile
  tamamlanır; düz HTTP server'ında bu `http://` demektir. Şeması olmayan ve `/`
  ile başlamayan bir hedef ise olduğu gibi gönderilir.
- Bu lab'daki sertifika self-signed'dır: hiçbir sertifika otoritesi ona kefil
  olmaz, bu yüzden istemcilere onu kabul etmeleri söylenmelidir (check `curl -k`
  kullanır).

Kaynak: [ngx_http_ssl_module](https://nginx.org/en/docs/http/ngx_http_ssl_module.html),
[Configuring HTTPS servers](https://nginx.org/en/docs/http/configuring_https_servers.html),
[return](https://nginx.org/en/docs/http/ngx_http_rewrite_module.html#return).

---

## Lab ortamı

Her şey bu dizinden build edilen tek bir Docker container içinde çalışır:

- `Dockerfile`, image'i `nginx:1.28.3-alpine` üzerine kurar, `openssl`'i yükler ve
  build sırasında `CN=localhost` için self-signed bir sertifika üretir:
  **/etc/nginx/certs/self.crt** (sertifika) ve **/etc/nginx/certs/self.key**
  (private key). Bunu değiştirmen gerekmez.
- `nginx.conf` senin dosyan. `http` bloğunda birkaç varsayılan dışında bir şey
  yok; TODO yorumları her server bloğunun neye ihtiyacı olduğunu listeler.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i
(`nginxlings/15_https_tls:check`) build eder, `nginx -T` çalıştırır, container'ın
80 portunu `127.0.0.1:18150`, 443 portunu `127.0.0.1:18450` adresine açar,
`Host: localhost` ile bir HTTP isteği gönderip başlıklarını okur, ardından `/`
adresini `curl -k` ile HTTPS üzerinden çeker.

---

## Görev

`nginx.conf` dosyasını düzenle ve `http` içine iki ayrı `server` bloğu ekle:

1. **80** portunda, `server_name localhost` olan ve her isteğe aynı host ve path'in
   `https://` URL'sine yönlendirmeyle cevap veren bir HTTP server.
2. **443** portunda TLS'i açık, `server_name localhost` olan, `Dockerfile`'ın
   ürettiği sertifika ve key'i kullanan ve `/` isteğine `tls-ok` ve ardından bir
   satır sonu içeren gövdeyle cevap veren bir HTTPS server.

---

## Başarı kriterleri

- Image build olmalı ve `nginx -t` yapılandırmayı kabul etmeli.
- Aktif yapılandırmada `listen 80` ve `listen 443 ssl` bulunmalı.
- 80 portu, `Location` başlığı `https://` ile başlayan bir yönlendirmeyle (301,
  302, 303, 307 veya 308) cevap vermeli.
- 443 portu TLS üzerinden `tls-ok` içeren bir gövdeyle cevap vermeli.
- Sertifika ve key, container içinde yukarıdaki path'lerde bulunmalı.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `No active 'listen 80;' yet`: başlangıç durumu bu. Önce HTTP server bloğunu ekle.
- `No active 'listen 443 ssl;' yet`: HTTPS bloğu eksik ya da `listen` satırında
  `ssl` parametresi yok.
- `unknown directive "ssl"`: `ssl on;` artık yok. TLS bunun yerine nerede açılıyor?
- `no "ssl_certificate" is defined for the "listen ... ssl" directive`: TLS
  server'ı hem sertifikaya hem key'e ihtiyaç duyar.
- `cannot load certificate "...": BIO_new_file() failed ... No such file`: path,
  `Dockerfile`'ın ürettiğiyle aynı değil. `openssl req` adımını oku.
- `cannot load certificate "...": PEM_read_bio_X509_AUX() failed`:
  `ssl_certificate`'e verilen dosya bir sertifika değil. Sertifika ile key yer mi
  değiştirmiş?
- `HTTP endpoint did not redirect to an https:// URL`: check'in yazdırdığı
  `Location` satırını oku. `http://...` ya da hiç şeması olmayan bir değer,
  yönlendirme hedefinin `https://` ile başlamadığını gösterir.
- `HTTPS endpoint did not return the expected body: tls-ok`: check'in yazdırdığı
  gövdeye bak. `404 Not Found` sayfası, HTTPS server'ının dönecek bir cevabı
  olmadığını gösterir. `301 Moved Permanently` sayfası ise yönlendirmenin 443
  portunda da çalıştığını gösterir: iki `listen` satırı aynı server bloğunda mı?

---

## Pekiştirme soruları

- Check neden `curl -k` kullanmak zorunda? Bir tarayıcı bu sertifika için ne
  gösterir ve tarayıcıların güvendiği bir sertifika almak için ne gerekir?
- Bir form, HTTP adresine `POST` gönderiyor. `301` yönlendirmesinde method'a ve
  gövdeye ne olabilir, `308` ile ne değişir?
- Yönlendirmende `$host` yerine `$server_name` kullan ve HTTP portuna
  `Host: shop.example` ile istek at. Yönlendirme şimdi nereyi gösteriyor, neden?
- Yapılandırmanda `ssl_protocols` satırı yok. Bu server hangi TLS sürümlerini kabul
  ediyor ve bunu container içinden nasıl kanıtlarsın?
- NGINX 1.28'de HTTPS server'ında HTTP/2'yi nasıl açarsın ve `nginx -t`,
  `listen 443 ssl http2;` için ne söyler?
