# 18 – OSI Layers & Use-Case Mapping (Theory + Mini Lab)

## Amaç

Bu egzersizde, NGINX’in farklı senaryolarda hangi rolde çalıştığını ayırt etmeyi öğreneceksin.

Odak noktaları:

- L4 (TCP/UDP) ile L7 (HTTP/HTTPS) arasındaki fark
- HTTP upstream kullandığında bunun neden L7 olduğu
- stream upstream kullandığında bunun neden L4 olduğu
- reverse proxy, load balancer, web server ve cache rollerini ayırmak

---

## Arkaplan

NGINX aynı üründe birden fazla rol üstlenebilir.

Örnekler:

- Statik dosya sunuyorsa → **web server**
- HTTP isteğini başka bir HTTP backend’e iletiyorsa → **L7 reverse proxy / L7 load balancer**
- TCP akışını başka bir TCP backend’e iletiyorsa → **L4 load balancer**
- Yanıtları saklayıp tekrar kullanıyorsa → **cache**

Buradaki önemli fark şudur:

### L7 örneği
NGINX HTTP katmanını bilir:
- request path
- headers
- host
- method
- cookies

Bu yüzden `http {}` içinde çalışan upstream yapısı **Layer 7** mantığıdır.

### L4 örneği
NGINX sadece TCP/UDP akışını taşır:
- HTTP header’larını yorumlamak zorunda değildir
- paket içeriğinin uygulama seviyesini bilmeden yönlendirme yapabilir

Bu yüzden `stream {}` içinde çalışan upstream yapısı **Layer 4** mantığıdır.

---

## Mini lab

Bu egzersizde iki yapı kuracaksın:

1. **HTTP / L7**
   - `/l7` isteği bir HTTP upstream grubuna gidecek
   - NGINX burada reverse proxy / L7 load balancer rolünde olacak

2. **TCP / L4**
   - `9000` portuna gelen ham TCP bağlantısı bir stream upstream grubuna gidecek
   - NGINX burada L4 load balancer rolünde olacak

---

## Görev

`nginx.conf` içinde:

- stream modülünü yükle
- bir `http` upstream tanımla
- bir `stream` upstream tanımla
- `/l7` endpoint’ini HTTP upstream’e proxy et
- `9000` portunu stream upstream’e yönlendir

HTTP tarafında ayrıca:
- cevabı veren backend’in adresini (`$upstream_addr`) `X-L7-Upstream` cevap
  header’ında görünür yap

Container’daki `start.sh`, NGINX’ten önce backend’leri başlatır:

- HTTP backend’leri: `127.0.0.1:8081` ve `127.0.0.1:8082`
- TCP backend’leri: `127.0.0.1:9001` ve `127.0.0.1:9002`

Image, Alpine’in `nginx-mod-stream` paketini kurar. Modül dosyası
`/usr/lib/nginx/modules/ngx_stream_module.so` yolundadır; `load_module` dosyanın
en üstünde, `events` bloğundan önce yer almalıdır.

---

## Beklenen sonuç

- `GET /l7` istekleri iki farklı HTTP backend arasında dağılmalı
- `9000` portuna yapılan TCP bağlantıları iki farklı TCP backend arasında dağılmalı
- Ayırt edebilmelisin:

  - `http upstream` → L7
  - `stream upstream` → L4
  - statik dosya sunumu → web server
  - cache yapılandırması olsaydı → cache rolü

---

## Başarı kriterleri

- Konfigürasyon geçerli olmalı
- `/l7` isteği çalışan bir HTTP upstream’e gitmeli
- `X-L7-Upstream` header’ı görünmeli
- `9000` portundaki TCP bağlantılarında iki backend de gözlenmeli
- `http` ve `stream` yapıları effective config içinde bulunmalı