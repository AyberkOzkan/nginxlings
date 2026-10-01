# 07 – HTTP Yük Dengeleyici (Katman 7)

## Amaç

Bu egzersizde NGINX'i istekleri iki arka uç sunucuya dağıtan bir **HTTP yük
dengeleyici (load balancer)** hâline getireceksin ve her isteği hangi arka uç
sunucunun yanıtladığını erişim günlüğünden (access log) kanıtlayacaksın.

Pratik edeceklerin:

- arka uç sunucuları (backend) bir `upstream` bloğunda gruplamak,
- istekleri `proxy_pass` ile bu gruba iletmek,
- yük dengeleme kararını `$upstream_addr` ile okumak.

---

## Arka plan – sunucu grubu ve sırayla dağıtım

`http` bağlamı (context) içindeki bir `upstream` bloğu, bir sunucu grubuna ad verir.
URL'sinin ana makine (host) kısmında bu adı kullanan bir `proxy_pass`, her isteği tek
bir adres yerine grubun üyelerinden birine gönderir.

Genel bir örnek (bu laboratuvardakinden farklı ad ve adresler):

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

- Varsayılan yöntem **sırayla dağıtım (round-robin)** yöntemidir: istekler sunuculara
  sırayla gider. `server` satırındaki `weight=` payı değiştirir; `upstream` bloğundaki
  `least_conn` ve `ip_hash` yöntemi değiştirir.
- `$upstream_addr`, yanıtı veren sunucunun adresini tutar. NGINX birden fazla
  sunucuyu denemek zorunda kaldıysa hepsini virgülle ayırarak listeler.
- `return`, isteği rewrite fazında, iletme işlemi başlamadan bitirir. İçinde hâlâ
  `return` bulunan bir `location` sunucu grubuna hiç ulaşmaz.

Kaynak: [ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html),
[proxy_pass](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass).

---

## Laboratuvar ortamı

Her şey bu dizinden oluşturulan (build) tek bir Docker konteyneri (container)
içinde çalışır:

- `start.sh`, konteyner içinde **127.0.0.1:8081** ve **127.0.0.1:8082** üzerinde iki
  küçük HTTP arka uç sunucusu başlatır, ardından NGINX'i ön planda (foreground)
  çalıştırır.
- `nginx.conf` senin dosyan. `upstream=$upstream_addr` içeren `lb_main` günlük biçimi
  ve erişim günlüğünün stdout'a yönlendirilmesi hazır durumda.
- `Dockerfile`, imajı (image) `nginx:1.28.3-alpine` üzerine kurar.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) imajı
(`nginxlings/07_http_load_balancer:check`) oluşturur, `nginx -t` çalıştırır,
konteynerin 8080 portunu `http://127.0.0.1:18080/` adresine açar, birkaç istek
gönderir ve konteyner günlüklerindeki `upstream=` değerlerini sayar.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `upstream backend` bloğundaki yer tutucu sunucuyu, `start.sh`'ın çalıştırdığı iki
   arka uç sunucuyla değiştir.
2. `location /` içindeki yer tutucu `return`'ü, istekler `backend` sunucu grubuna
   gidecek şekilde değiştir.
3. `log_format lb_main` ve `access_log` satırlarını koru; kontrol bunların ürettiği
   `upstream=` alanını okur.

---

## Başarı kriterleri

- İmaj oluşturulmalı ve `nginx -t` yapılandırmayı kabul etmeli.
- 8080 portuna gelen istekler sunucu grubu üzerinden yanıtlanmalı.
- Erişim günlüğünde en az bir isteği `127.0.0.1:8081`, en az bir isteği
  `127.0.0.1:8082` yanıtlamış görünmeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- Günlük çıktısında `upstream=127.0.0.1:65535` var: yer tutucu sunucu hâlâ `upstream`
  bloğunda.
- `upstream=-` görünüyor: istek hiçbir sunucuya ulaşmamış. `location /` içinde
  `return` hâlâ duruyor mu?
- Yalnızca bir arka uç sunucu görünüyor: `upstream` bloğundaki `server` satırlarını
  say ve iki portun da `start.sh`'ın başlattıklarıyla aynı olduğunu kontrol et.
- `nginx -t` `invalid URL prefix` diyor: `proxy_pass` şemalı bir URL bekler ve URL'nin
  ana makine kısmı sunucu grubunun adıdır.

---

## Pekiştirme soruları

- İlk sunucuya `weight=3` eklersen dağılım nasıl görünür?
- Bir istemciyi her zaman aynı arka uç sunucuya göndermek için `upstream` bloğuna
  hangi direktifi eklersin ve bu direktif anahtar olarak neyi kullanır?
- İlk arka uç sunucu bağlantıyı reddedip NGINX ikincisini denediğinde
  `$upstream_addr` ne içerir? (Egzersiz 10 buna dayanır.)
