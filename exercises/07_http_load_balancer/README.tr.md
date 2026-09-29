# 07 – HTTP Load Balancer (Layer 7)

## Amaç

Bu egzersizde NGINX'i istekleri iki backend sunucuya dağıtan bir **HTTP load
balancer** hâline getireceksin ve her isteği hangi backend'in cevapladığını access
log'dan kanıtlayacaksın.

Pratik edeceklerin:

- backend sunucuları bir `upstream` bloğunda gruplamak,
- istekleri `proxy_pass` ile bu gruba iletmek,
- load balancing kararını `$upstream_addr` ile okumak.

---

## Arka plan – upstream ve round-robin

`http` context'i içindeki bir `upstream` bloğu, bir sunucu grubuna ad verir. URL'sinin
host kısmında bu adı kullanan bir `proxy_pass`, her isteği tek bir adres yerine
grubun üyelerinden birine gönderir.

Genel bir örnek (bu lab'dakinden farklı ad ve adresler):

    upstream app_pool {
        server 10.0.0.11:3000;
        server 10.0.0.12:3000;
    }

    server {
        listen 80;

        location /api/ {
            proxy_pass http://app_pool;
        }
    }

Bilinmesi gerekenler:

- Varsayılan yöntem **round-robin**'dir: istekler sunuculara sırayla gider.
  `server` satırındaki `weight=` payı değiştirir; `upstream` bloğundaki `least_conn`
  ve `ip_hash` yöntemi değiştirir.
- `$upstream_addr`, cevabı veren sunucunun adresini tutar. NGINX birden fazla
  sunucuyu denemek zorunda kaldıysa hepsini virgülle ayırarak listeler.
- `return`, isteği rewrite fazında, proxy işlemi başlamadan bitirir. İçinde hâlâ
  `return` bulunan bir location upstream'e hiç ulaşmaz.

Kaynak: [ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html),
[proxy_pass](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass).

---

## Lab ortamı

Her şey bu dizinden build edilen tek bir Docker container içinde çalışır:

- `start.sh`, container içinde **127.0.0.1:8081** ve **127.0.0.1:8082** üzerinde iki
  küçük HTTP backend başlatır, ardından NGINX'i foreground'da çalıştırır.
- `nginx.conf` senin dosyan. `upstream=$upstream_addr` içeren `lb_main` log formatı ve
  access log'un stdout'a yönlendirilmesi hazır durumda.
- `Dockerfile`, image'i `nginx:1.28.3-alpine` üzerine kurar.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i
(`nginxlings/07_http_load_balancer:check`) build eder, `nginx -t` çalıştırır,
container'ın 8080 portunu `http://127.0.0.1:18080/` adresine açar, birkaç istek
gönderir ve container log'larındaki `upstream=` değerlerini sayar.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `upstream backend` bloğundaki yer tutucu sunucuyu, `start.sh`'ın çalıştırdığı iki
   backend ile değiştir.
2. `location /` içindeki yer tutucu `return`'ü, istekler `backend` upstream grubuna
   gidecek şekilde değiştir.
3. `log_format lb_main` ve `access_log` satırlarını koru; check bunların ürettiği
   `upstream=` alanını okur.

---

## Başarı kriterleri

- Image build olmalı ve `nginx -t` yapılandırmayı kabul etmeli.
- 8080 portuna gelen istekler upstream grubu üzerinden cevaplanmalı.
- Access log'da en az bir isteği `127.0.0.1:8081`, en az bir isteği `127.0.0.1:8082`
  cevaplamış görünmeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- Log çıktısında `upstream=127.0.0.1:65535` var: yer tutucu sunucu hâlâ upstream
  bloğunda.
- `upstream=-` görünüyor: istek hiçbir upstream'e ulaşmamış. `location /` içinde
  `return` hâlâ duruyor mu?
- Yalnızca bir backend görünüyor: upstream bloğundaki `server` satırlarını say ve iki
  portun da `start.sh`'ın başlattıklarıyla aynı olduğunu kontrol et.
- `nginx -t` `invalid URL prefix` diyor: `proxy_pass` şemalı bir URL bekler ve URL'nin
  host kısmı upstream'in adıdır.

---

## Pekiştirme soruları

- İlk sunucuya `weight=3` eklersen dağılım nasıl görünür?
- Bir istemciyi her zaman aynı backend'e göndermek için `upstream` bloğuna hangi
  direktifi eklersin ve bu direktif anahtar olarak neyi kullanır?
- İlk backend bağlantıyı reddedip NGINX ikincisini denediğinde `$upstream_addr`
  ne içerir? (Egzersiz 10 buna dayanır.)
