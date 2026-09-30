# 08 – TCP Load Balancer (Layer 4, stream)

## Amaç

Bu egzersizde NGINX'i, ham TCP bağlantılarını iki backend'e dağıtan bir **Layer 4
(TCP) load balancer** hâline getireceksin. Bunun için `http` yerine `stream`
context'ini kullanacaksın.

Pratik edeceklerin:

- `stream` bloğunu `nginx.conf` içinde doğru seviyeye yerleştirmek,
- TCP backend'lerini `stream` içindeki bir `upstream` bloğunda gruplamak,
- bağlantıları bütünüyle bu gruba, `proxy_pass`'in stream biçimiyle iletmek.

---

## Arka plan – http ve stream

NGINX protokolleri ayrı üst düzey context'lerde işler:

- `http`, HTTP isteklerini anlar (Layer 7): method, header, URI, `location`.
- `stream` yalnızca TCP veya UDP bağlantılarını görür (Layer 4): gelen ve giden
  byte'lar.

`stream`, `nginx.conf`'un en üst seviyesinde `http`'nin yanında durur, asla onun
içinde değil. Her context'in kendi `upstream` ve `server` blokları vardır.

Redis bağlantılarını dağıtan genel bir örnek (bu lab'dakinden farklı ad ve
adresler):

    stream {
        upstream redis_pool {
            server 10.0.0.31:6379;
            server 10.0.0.32:6379;
        }

        server {
            listen 16379;
            proxy_pass redis_pool;
        }
    }

Bilinmesi gerekenler:

- Stream `server` bloğunda `location` yoktur. `proxy_pass` doğrudan `server`'ın
  içine yazılır ve `http://` gibi bir şema **olmadan** bir adres ya da upstream adı
  alır.
- HTTP istekleri üzerinde çalışan direktifler (`location`, `proxy_set_header`,
  `add_header`, ...) `stream`'de yoktur: üzerinde işlem yapılacak bir istek yoktur.
- `server_name` 1.25.5'ten beri `stream`'de de vardır, ama HTTP `Host` başlığıyla
  değil, TLS Server Name Indication (SNI) bilgisiyle eşleşir.
- Varsayılan yöntem, `http`'de olduğu gibi round-robin'dir. Stream `upstream`
  bloğunda `weight=`, `least_conn`, `hash` ve `random` bunu değiştirir.
- `stream` UDP trafiğini de proxy edebilir (`listen ... udp`); bu lab yalnızca TCP
  kullanır.

Kaynak: [ngx_stream_core_module](https://nginx.org/en/docs/stream/ngx_stream_core_module.html),
[ngx_stream_proxy_module](https://nginx.org/en/docs/stream/ngx_stream_proxy_module.html),
[ngx_stream_upstream_module](https://nginx.org/en/docs/stream/ngx_stream_upstream_module.html).

---

## Lab ortamı

Her şey bu dizinden build edilen tek bir Docker container içinde çalışır:

- `start.sh`, container içinde `socat` ile iki TCP backend başlatır, ardından
  NGINX'i foreground'da çalıştırır. **127.0.0.1:9001** her bağlantıya
  `backend=9001` satırıyla, **127.0.0.1:9002** ise `backend=9002` satırıyla cevap
  verir.
- `nginx.conf` senin dosyan. Bir `upstream tcp_backend` bloğu ve 9000 portunda bir
  `server` ile başlar; ikisi de `http` içindedir. Yanında isteğe bağlı bir log
  formatı vardır.
- `Dockerfile`, image'i `nginx:1.28.3-alpine` üzerine kurar ve `socat`'ı yükler.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i
(`nginxlings/08_stream_load_balancer:check`) build eder, `nginx -t` ve `nginx -T`
çalıştırır, container'ın 9000 portunu `127.0.0.1:19000` adresine açar, Python 3 ile
10 TCP bağlantısı kurar ve her birinin döndürdüğü ilk satırı okur.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `tcp_backend` upstream'ini, `start.sh`'ın çalıştırdığı iki backend ile doldur.
2. Proxy işlemini Layer 4'e taşı: upstream ve **9000** portunu dinleyen server, üst
   seviyedeki bir `stream` bloğunda olmalı. Yanına ikinci bir blok eklemek yerine
   başlangıçtaki `http` bloğunu dönüştür.
3. Bu server'da her bağlantıyı upstream grubuna ilet.
4. `log_format` ve `access_log` satırları isteğe bağlıdır. Tutarsan `stream`
   içinde de çalışırlar.

---

## Başarı kriterleri

- Image build olmalı ve `nginx -t` yapılandırmayı kabul etmeli.
- `nginx -T` üst seviyede bir `stream` bloğu ve aktif bir `proxy_pass` göstermeli.
- Container başladıktan sonra 20 saniye içinde 9000 portu backend verisi dönmeli.
- 10 bağlantının en az biri `backend=9001`, en az biri `backend=9002` dönmeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `no servers are inside upstream`: upstream bloğu hâlâ boş.
- `"proxy_pass" directive is not allowed here`: server hâlâ `http` içinde; orada
  `proxy_pass` bir `location`'a yazılır. Layer 4 proxy için `stream` gerekir.
- `"stream" directive is not allowed here`: `stream`, `http`'nin içine yazılmış.
  `events` ve `http` gibi dosyanın en üst seviyesinde olmalı.
- `"location" directive is not allowed here` veya
  `"proxy_set_header" directive is not allowed here`: bunlar HTTP direktifleri.
  `stream`'de bağlantıyı `server`'ın kendisi iletir.
- `invalid host in upstream "http://..."`: stream `proxy_pass` şemasız, yalın bir
  adres ya da upstream adı alır.
- `invalid URL prefix`: `proxy_pass` bir `http` location'ı içinde; bu Layer 7.
- `The container exited during startup` ve log'larda
  `bind() to 0.0.0.0:9000 failed (98: Address in use)`: iki blok birden 9000
  portunu dinliyor. Eski `http` server'ı `stream` bloğunun yanında hâlâ duruyor mu?
- `Got backend=9001 count=10, backend=9002 count=0` (ya da tersi): upstream
  yalnızca bir backend'e ulaşıyor. `server` satırlarını say ve portları `start.sh`
  ile karşılaştır.

---

## Pekiştirme soruları

- Hem `http`'nin hem `stream`'in 9000 portunu dinlediği bir yapılandırma
  `nginx -t`'den geçiyor, ama NGINX başlayamıyor. Test bunu neden yakalamıyor?
- Başlangıçtaki log formatı `status=$status` yazıyor, ama `stream` HTTP cevabı
  göndermez. Orada `$status` ne anlama gelir ve backend'i kapalı olan bir bağlantı
  için log'da ne görürsün?
- Egzersiz 07'de bir istemciyi hep aynı backend'e göndermek için kullanacağın
  direktif stream `upstream` bloğunda çalışır mı? Çalışmıyorsa yerine ne
  kullanırsın?
- Aynı portu dinleyen, farklı `server_name` değerlerine sahip iki stream `server`
  bloğunun bağlantıları farklı backend'lere gönderebilmesi için neye ihtiyacı
  vardır? Bu, bu lab'daki düz TCP backend'lerinde neden işe yaramaz?
