# 08 – TCP Yük Dengeleyici (Katman 4, stream)

## Amaç

Bu egzersizde NGINX'i, ham TCP bağlantılarını iki arka uç sunucuya dağıtan bir
**Katman 4 (TCP) yük dengeleyici (load balancer)** hâline getireceksin. Bunun için
`http` yerine `stream` bağlamını (context) kullanacaksın.

Pratik edeceklerin:

- `stream` bloğunu `nginx.conf` içinde doğru seviyeye yerleştirmek,
- TCP arka uç sunucularını (backend) `stream` içindeki bir `upstream` bloğunda
  gruplamak,
- bağlantıları bütünüyle bu gruba, `proxy_pass`'in stream biçimiyle iletmek.

---

## Arka plan – http ve stream

NGINX protokolleri ayrı üst düzey bağlamlarda işler:

- `http`, HTTP isteklerini anlar (Katman 7): yöntem, başlık (header), URI, `location`.
- `stream` yalnızca TCP veya UDP bağlantılarını görür (Katman 4): gelen ve giden
  baytlar.

`stream`, `nginx.conf`'un en üst seviyesinde `http`'nin yanında durur, asla onun
içinde değil. Her bağlamın kendi `upstream` ve `server` blokları vardır.

Redis bağlantılarını dağıtan genel bir örnek (bu laboratuvardakinden farklı ad ve
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
  içine yazılır ve `http://` gibi bir şema **olmadan** bir adres ya da sunucu grubu
  adı alır.
- HTTP istekleri üzerinde çalışan direktifler (`location`, `proxy_set_header`,
  `add_header`, ...) `stream`'de yoktur: üzerinde işlem yapılacak bir istek yoktur.
- `server_name` 1.25.5'ten beri `stream`'de de vardır, ama HTTP `Host` başlığıyla
  değil, TLS Server Name Indication (SNI) bilgisiyle eşleşir.
- Varsayılan yöntem, `http`'de olduğu gibi sırayla dağıtımdır (round-robin). Stream
  `upstream` bloğunda `weight=`, `least_conn`, `hash` ve `random` bunu değiştirir.
- `stream` UDP trafiğini de iletebilir (`listen ... udp`); bu laboratuvar yalnızca
  TCP kullanır.

Kaynak: [ngx_stream_core_module](https://nginx.org/en/docs/stream/ngx_stream_core_module.html),
[ngx_stream_proxy_module](https://nginx.org/en/docs/stream/ngx_stream_proxy_module.html),
[ngx_stream_upstream_module](https://nginx.org/en/docs/stream/ngx_stream_upstream_module.html).

---

## Laboratuvar ortamı

Her şey bu dizinden oluşturulan (build) tek bir Docker konteyneri (container)
içinde çalışır:

- `start.sh`, konteyner içinde `socat` ile iki TCP arka uç sunucusu başlatır, ardından
  NGINX'i ön planda (foreground) çalıştırır. **127.0.0.1:9001** her bağlantıya
  `backend=9001` satırıyla, **127.0.0.1:9002** ise `backend=9002` satırıyla yanıt
  verir.
- `nginx.conf` senin dosyan. Bir `upstream tcp_backend` bloğu ve 9000 portunda bir
  `server` ile başlar; ikisi de `http` içindedir. Yanında isteğe bağlı bir günlük
  biçimi (log format) vardır.
- `Dockerfile`, imajı (image) `nginx:1.28.3-alpine` üzerine kurar ve `socat`'ı
  yükler.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) imajı
(`nginxlings/08_stream_load_balancer:check`) oluşturur, `nginx -t` ve `nginx -T`
çalıştırır, konteynerin 9000 portunu `127.0.0.1:19000` adresine açar, Python 3 ile
10 TCP bağlantısı kurar ve her birinin döndürdüğü ilk satırı okur.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `tcp_backend` sunucu grubunu, `start.sh`'ın çalıştırdığı iki arka uç sunucuyla
   doldur.
2. İletme işlemini Katman 4'e taşı: `upstream` ve **9000** portunu dinleyen `server`,
   üst seviyedeki bir `stream` bloğunda olmalı. Yanına ikinci bir blok eklemek yerine
   başlangıçtaki `http` bloğunu dönüştür.
3. Bu `server`'da her bağlantıyı sunucu grubuna ilet.
4. `log_format` ve `access_log` satırları isteğe bağlıdır. Tutarsan `stream`
   içinde de çalışırlar.

---

## Başarı kriterleri

- İmaj oluşturulmalı ve `nginx -t` yapılandırmayı kabul etmeli.
- `nginx -T` üst seviyede bir `stream` bloğu ve aktif bir `proxy_pass` göstermeli.
- Konteyner başladıktan sonra 20 saniye içinde 9000 portu arka uç verisi dönmeli.
- 10 bağlantının en az biri `backend=9001`, en az biri `backend=9002` dönmeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `no servers are inside upstream`: `upstream` bloğu hâlâ boş.
- `"proxy_pass" directive is not allowed here`: `server` hâlâ `http` içinde; orada
  `proxy_pass` bir `location`'a yazılır. Katman 4'te iletme için `stream` gerekir.
- `"stream" directive is not allowed here`: `stream`, `http`'nin içine yazılmış.
  `events` ve `http` gibi dosyanın en üst seviyesinde olmalı.
- `"location" directive is not allowed here` veya
  `"proxy_set_header" directive is not allowed here`: bunlar HTTP direktifleri.
  `stream`'de bağlantıyı `server`'ın kendisi iletir.
- `invalid host in upstream "http://..."`: stream `proxy_pass` şemasız, yalın bir
  adres ya da sunucu grubu adı alır.
- `invalid URL prefix`: `proxy_pass` bir `http` `location`'ı içinde; bu Katman 7.
- `The container exited during startup` ve günlüklerde
  `bind() to 0.0.0.0:9000 failed (98: Address in use)`: iki blok birden 9000
  portunu dinliyor. Eski `http` `server`'ı `stream` bloğunun yanında hâlâ duruyor mu?
- `Got backend=9001 count=10, backend=9002 count=0` (ya da tersi): sunucu grubu
  yalnızca bir arka uç sunucuya ulaşıyor. `server` satırlarını say ve portları
  `start.sh` ile karşılaştır.

---

## Pekiştirme soruları

- Hem `http`'nin hem `stream`'in 9000 portunu dinlediği bir yapılandırma
  `nginx -t`'den geçiyor, ama NGINX başlayamıyor. Test bunu neden yakalamıyor?
- Başlangıçtaki günlük biçimi `status=$status` yazıyor, ama `stream` HTTP yanıtı
  göndermez. Orada `$status` ne anlama gelir ve arka uç sunucusu kapalı olan bir
  bağlantı için günlükte ne görürsün?
- Egzersiz 07'de bir istemciyi hep aynı arka uç sunucuya göndermek için kullanacağın
  direktif stream `upstream` bloğunda çalışır mı? Çalışmıyorsa yerine ne
  kullanırsın?
- Aynı portu dinleyen, farklı `server_name` değerlerine sahip iki stream `server`
  bloğunun bağlantıları farklı arka uç sunuculara gönderebilmesi için neye ihtiyacı
  vardır? Bu, bu laboratuvardaki düz TCP arka uç sunucularında neden işe yaramaz?
