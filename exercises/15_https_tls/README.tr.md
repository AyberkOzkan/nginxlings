# 15 – HTTPS (TLS) Hızlı Uygulama

## Amaç

Bu egzersizde NGINX'e **TLS sonlandırma** (TLS termination) yaptıracaksın: imaj
(image) içinde hazır bulunan bir sertifikayla HTTPS üzerinden yanıt verecek ve düz
HTTP istemcilerini HTTPS adresine yönlendirecek.

Pratik edeceklerin:

- dinlenen bir soket (socket) üzerinde TLS'i açmak,
- NGINX'e bir sertifikayı ve onun özel anahtarını (private key) göstermek,
- ana makine adını (host name) ve yolu kaybetmeden HTTP'yi HTTPS'e yönlendirmek.

---

## Arka plan – TLS sonlandırma ve HTTP yönlendirmesi

TLS sonlandırmada NGINX istemciyle TLS el sıkışmasını (handshake) yapar, trafiği
çözer ve içindeki düz HTTP isteğini işler. Tipik bir site iki `server` bloğu
kullanır: biri şifreli bağlantıları kabul eder, düz HTTP portundaki diğeri ise
istemcilere yalnızca HTTPS üzerinden tekrar gelmelerini söyler.

Genel bir örnek (bu laboratuvardakinden farklı ad, port ve dosyalar):

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
  `ssl_certificate_key` ise özel anahtarı gösterir. İkisi de zorunludur ve
  `nginx -t` dosyaları yükler; bu yüzden yanlış bir yol ilk istekte değil, testte
  hata verir.
- `server` bloğundaki `return code URL;`, herhangi bir `location` seçilmeden önce
  yanıt verir. `$host` istemcinin istediği ana makine adını, `$request_uri` ise orijinal
  yolu ve sorgu dizesini (query string) korur.
- `/` ile başlayan bir yönlendirme (redirect) hedefi, mevcut isteğin şeması ve ana
  makine adı ile tamamlanır; düz HTTP sunucusunda bu `http://` demektir. Şeması
  olmayan ve `/` ile başlamayan bir hedef ise olduğu gibi gönderilir.
- Bu laboratuvardaki sertifika kendinden imzalıdır (self-signed): hiçbir sertifika
  otoritesi ona kefil olmaz, bu yüzden istemcilere onu kabul etmeleri söylenmelidir
  (kontrol `curl -k` kullanır).

Kaynak: [ngx_http_ssl_module](https://nginx.org/en/docs/http/ngx_http_ssl_module.html),
[Configuring HTTPS servers](https://nginx.org/en/docs/http/configuring_https_servers.html),
[return](https://nginx.org/en/docs/http/ngx_http_rewrite_module.html#return).

---

## Laboratuvar ortamı

Her şey bu dizinden oluşturulan (build) tek bir Docker konteyneri (container)
içinde çalışır:

- `Dockerfile`, imajı `nginx:1.28.3-alpine` üzerine kurar, `openssl`'i yükler ve
  oluşturma sırasında `CN=localhost` için kendinden imzalı bir sertifika üretir:
  **/etc/nginx/certs/self.crt** (sertifika) ve **/etc/nginx/certs/self.key**
  (özel anahtar). Bunu değiştirmen gerekmez.
- `nginx.conf` senin dosyan. `http` bloğunda birkaç varsayılan dışında bir şey
  yok; TODO yorumları her `server` bloğunun neye ihtiyacı olduğunu listeler.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) imajı
(`nginxlings/15_https_tls:check`) oluşturur, `nginx -T` çalıştırır, konteynerin
80 portunu `127.0.0.1:18150`, 443 portunu `127.0.0.1:18450` adresine açar,
`Host: localhost` ile bir HTTP isteği gönderip başlıklarını (header) okur,
ardından `/` adresini `curl -k` ile HTTPS üzerinden çeker.

---

## Görev

`nginx.conf` dosyasını düzenle ve `http` içine iki ayrı `server` bloğu ekle:

1. **80** portunda, `server_name localhost` olan ve her isteğe aynı ana makine adı ve
   yolun `https://` URL'sine yönlendirmeyle yanıt veren bir HTTP sunucusu.
2. **443** portunda TLS'i açık, `server_name localhost` olan, `Dockerfile`'ın
   ürettiği sertifika ve anahtarı kullanan ve `/` isteğine `tls-ok` ve ardından bir
   satır sonu içeren gövdeyle yanıt veren bir HTTPS sunucusu.

---

## Başarı kriterleri

- İmaj oluşturulmalı ve `nginx -t` yapılandırmayı kabul etmeli.
- Aktif yapılandırmada `listen 80` ve `listen 443 ssl` bulunmalı.
- 80 portu, `Location` başlığı `https://` ile başlayan bir yönlendirmeyle (301,
  302, 303, 307 veya 308) yanıt vermeli.
- 443 portu TLS üzerinden `tls-ok` içeren bir gövdeyle yanıt vermeli.
- Sertifika ve anahtar, konteyner içinde yukarıdaki yollarda bulunmalı.

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `No active 'listen 80;' yet`: başlangıç durumu bu. Önce HTTP `server` bloğunu ekle.
- `No active 'listen 443 ssl;' yet`: HTTPS bloğu eksik ya da `listen` satırında
  `ssl` parametresi yok.
- `unknown directive "ssl"`: `ssl on;` artık yok. TLS bunun yerine nerede açılıyor?
- `no "ssl_certificate" is defined for the "listen ... ssl" directive`: TLS
  sunucusu hem sertifikaya hem anahtara ihtiyaç duyar.
- `cannot load certificate "...": BIO_new_file() failed ... No such file`: yol,
  `Dockerfile`'ın ürettiğiyle aynı değil. `openssl req` adımını oku.
- `cannot load certificate "...": PEM_read_bio_X509_AUX() failed`:
  `ssl_certificate`'e verilen dosya bir sertifika değil. Sertifika ile anahtar yer
  mi değiştirmiş?
- `HTTP endpoint did not redirect to an https:// URL`: kontrolün yazdırdığı
  `Location` satırını oku. `http://...` ya da hiç şeması olmayan bir değer,
  yönlendirme hedefinin `https://` ile başlamadığını gösterir.
- `HTTPS endpoint did not return the expected body: tls-ok`: kontrolün yazdırdığı
  gövdeye bak. `404 Not Found` sayfası, HTTPS sunucusunun dönecek bir yanıtı
  olmadığını gösterir. `301 Moved Permanently` sayfası ise yönlendirmenin 443
  portunda da çalıştığını gösterir: iki `listen` satırı aynı `server` bloğunda mı?

---

## Pekiştirme soruları

- Kontrol neden `curl -k` kullanmak zorunda? Bir tarayıcı bu sertifika için ne
  gösterir ve tarayıcıların güvendiği bir sertifika almak için ne gerekir?
- Bir form, HTTP adresine `POST` gönderiyor. `301` yönlendirmesinde yönteme ve
  gövdeye ne olabilir, `308` ile ne değişir?
- Yönlendirmende `$host` yerine `$server_name` kullan ve HTTP portuna
  `Host: shop.example` ile istek at. Yönlendirme şimdi nereyi gösteriyor, neden?
- Yapılandırmanda `ssl_protocols` satırı yok. Bu sunucu hangi TLS sürümlerini kabul
  ediyor ve bunu konteyner içinden nasıl kanıtlarsın?
- NGINX 1.28'de HTTPS sunucusunda HTTP/2'yi nasıl açarsın ve `nginx -t`,
  `listen 443 ssl http2;` için ne söyler?
