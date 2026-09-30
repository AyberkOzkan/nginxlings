# 18 – OSI Layers & Use-Case Mapping (Theory + Mini Lab)

## Amaç

Bu egzersizde NGINX'in bir senaryoda oynadığı rolü ve çalıştığı katmanı
adlandıracak, aynı NGINX içinde bir Layer 7 ve bir Layer 4 yolu kuracaksın.

Pratik edeceklerin:

- Layer 7 (HTTP) kullanımlarını Layer 4 (TCP/UDP) kullanımlarından ayırmak,
- `load_module` ile dinamik bir modül yüklemek,
- bir `http` upstream'ini ve bir `stream` upstream'ini yan yana çalıştırmak.

---

## Arka plan – roller ve katmanlar

Aynı NGINX programı, yapılandırmasına göre farklı roller oynar:

| Yapılandırma | Rol | Katman |
| --- | --- | --- |
| Dosya sunan `root` / `index` | web sunucusu | L7 |
| `http` içinde `upstream` + `proxy_pass http://...` | reverse proxy, L7 load balancer | L7 |
| Upstream cevaplarını saklayan `proxy_cache` | önbellek (cache) | L7 |
| `stream` içinde `upstream` + `proxy_pass ad` | L4 load balancer | L4 |

**Layer 7**'de NGINX HTTP'yi çözümler: method, URI, başlıklar, host, cookie'ler.
Path'e göre yönlendirebilir, başlık ekleyebilir, cevapları önbelleğe alabilir.
`http {}` içindeki bir upstream'in L7 kullanımı olmasının sebebi budur (egzersiz 07).

**Layer 4**'te NGINX yalnızca TCP ya da UDP bağlantılarını iletir. Byte'ları
yorumlamaz; üzerinde işlem yapılacak URI ya da başlık yoktur. `stream {}` içindeki bir
upstream'in L4 kullanımı olmasının sebebi budur (egzersiz 08).

Bilinmesi gerekenler:

- NGINX modülleri ya programın içine gömülü derlenir ya da `load_module <path>;`
  ile yüklenen **dinamik modüller** (`.so` dosyaları) olarak derlenir.
  `load_module`, `events` bloğundan önce, `main` context'e aittir.
- `nginx -V` bir modülün nasıl derlendiğini gösterir: `--with-stream` gömülü,
  `--with-stream=dynamic` ise yüklenmesi gerektiği anlamına gelir.
- `http` ve `stream` ayrı üst düzey context'lerdir; her birinin kendi `upstream` ve
  `server` blokları vardır.

Kaynak: [load_module](https://nginx.org/en/docs/ngx_core_module.html#load_module),
[ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html),
[ngx_stream_upstream_module](https://nginx.org/en/docs/stream/ngx_stream_upstream_module.html).

---

## Lab ortamı

Her şey bu dizinden build edilen tek bir Docker container içinde çalışır:

- `Dockerfile`, image'i `alpine:3.20.10` üzerine kurar; Alpine'in `nginx` ve
  `nginx-mod-stream` paketlerini, backend'ler için de `python3` ve `socat`'ı yükler.
  Stream modülünün dosyası **/usr/lib/nginx/modules/ngx_stream_module.so**'dur.
- `start.sh` önce backend'leri, sonra NGINX'i başlatır:
  - HTTP: **127.0.0.1:8081** `l7-backend=8081`, **127.0.0.1:8082**
    `l7-backend=8082` cevabı verir.
  - TCP: **127.0.0.1:9001** `l4-backend=9001`, **127.0.0.1:9002**
    `l4-backend=9002` cevabı verir.
- `nginx.conf` senin dosyan. 8080 portundaki server'ın `location /l7` bloğu hâlâ yer
  tutucu metin döndürüyor; TODO yorumları modülü ve stream kısmını işaretler.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i
(`nginxlings/18_osi_layers_usecases:check`) build eder, aktif yapılandırmayı
`nginx -T` ile okur, 8080 portunu `127.0.0.1:18180`, 9000 portunu `127.0.0.1:19080`
adresine açar, iki HTTP backend'i de `X-L7-Upstream` başlığında görünene kadar `/l7`
ister, ardından iki TCP backend'i de cevap verene kadar 9000 portuna TCP bağlantıları
açar.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. Stream modülünü dosyanın en üstünde yükle.
2. `http` içinde iki HTTP backend'i içeren bir upstream tanımla ve `location /l7`
   metin döndürmek yerine ona proxy etsin.
3. `location /l7` içinde, cevabı veren backend'in adresini taşıyan
   **X-L7-Upstream** response header'ını ekle.
4. İki TCP backend'ini içeren bir upstream ve bağlantıları ona ileten, **9000**
   portunu dinleyen bir server ile üst düzey bir `stream` bloğu ekle.

---

## Başarı kriterleri

- `nginx -T`, `ngx_stream_module.so` için `load_module` satırını, bir `stream`
  bloğunu ve `X-L7-Upstream` başlığını göstermeli.
- `/l7` cevaplarında `X-L7-Upstream` içinde hem `:8081` hem `:8082` görünmeli.
- 9000 portuna yapılan TCP bağlantıları hem `l4-backend=9001`'e hem
  `l4-backend=9002`'ye ulaşmalı.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `The stream module is not loaded yet`: başlangıç durumu bu.
- `NGINX does not know the stream directive yet`: `stream` bloğu yazılmış ama onu
  sağlayan modül yüklenmemiş.
- `"load_module" directive is specified too late`: `events` bloğundan sonra duruyor.
  `main` context nerede başlar?
- `"stream" directive is not allowed here`: `stream`, `http`'nin içine yazılmış
  (bkz. egzersiz 08).
- `No X-L7-Upstream header yet`: başlık yok ya da yorum satırında.
- `/l7 answered without an X-L7-Upstream value`: `/l7`'ye proxy'den önce hâlâ bir
  `return` cevap veriyor ya da başlık `/l7`'ye uygulanmayan bir yerde tanımlı
  (`add_header`'ın nasıl miras alındığını hatırla, egzersiz 03).
- `Got 8081=60, 8082=0` ya da `Got 9001=40, 9002=0`: o upstream yalnızca bir
  backend'e ulaşıyor. `server` satırlarını say ve portları `start.sh` ile
  karşılaştır.

---

## Pekiştirme soruları

- Her senaryo için rolü ve katmanı adlandır: NGINX TLS'i sonlandırıp HTTP uygulama
  sunucularına proxy ediyor; NGINX PostgreSQL bağlantılarını iki replikaya
  dağıtıyor; NGINX backend cevaplarını `proxy_cache` ile saklıyor; NGINX TLS
  bağlantılarını çözmeden SNI'ye göre farklı backend'lere gönderiyor.
- Stream tarafında `X-L7-Upstream` başlığı yok. Bir bağlantıya hangi TCP
  backend'inin hizmet verdiğini nerede görebilirdin?
- Egzersiz 08 `stream`'i hiç `load_module` olmadan kullandı. Orada neden çalıştı da
  burada çalışmıyor?
- Göreli path'li `load_module modules/ngx_stream_module.so;` de bu image'da çalışır.
  Neye göre göreli?
