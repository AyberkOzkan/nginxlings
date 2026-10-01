# 18 – OSI Katmanları ve Kullanım Senaryoları

## Amaç

Bu egzersizde NGINX'in bir senaryoda oynadığı rolü ve çalıştığı katmanı
adlandıracak, aynı NGINX içinde bir Katman 7 (Layer 7) ve bir Katman 4 (Layer 4)
yolu kuracaksın.

Pratik edeceklerin:

- Katman 7 (HTTP) kullanımlarını Katman 4 (TCP/UDP) kullanımlarından ayırmak,
- `load_module` ile dinamik bir modül (dynamic module) yüklemek,
- bir `http` `upstream`'ini ve bir `stream` `upstream`'ini yan yana çalıştırmak.

---

## Arka plan – roller ve katmanlar

Aynı NGINX programı, yapılandırmasına göre farklı roller oynar:

- Dosya sunan `root` / `index`: **web sunucusu** (L7)
- `http` içinde `upstream` + `proxy_pass http://...`: **ters vekil sunucu (reverse proxy), L7 yük dengeleyici (load balancer)** (L7)
- Arka uç yanıtlarını saklayan `proxy_cache`: **önbellek (cache)** (L7)
- `stream` içinde `upstream` + `proxy_pass ad`: **L4 yük dengeleyici** (L4)

**Katman 7**'de NGINX HTTP'yi çözümler: yöntem, URI, başlıklar (header), ana
makine adı, çerezler. Yola göre yönlendirebilir, başlık ekleyebilir, yanıtları
önbelleğe alabilir. `http {}` içindeki bir `upstream`'in L7 kullanımı olmasının
sebebi budur (egzersiz 07).

**Katman 4**'te NGINX yalnızca TCP ya da UDP bağlantılarını iletir. Baytları
yorumlamaz; üzerinde işlem yapılacak URI ya da başlık yoktur. `stream {}` içindeki
bir `upstream`'in L4 kullanımı olmasının sebebi budur (egzersiz 08).

Bilinmesi gerekenler:

- NGINX modülleri ya programın içine gömülü derlenir ya da `load_module <path>;`
  ile yüklenen **dinamik modüller** (`.so` dosyaları) olarak derlenir.
  `load_module`, `events` bloğundan önce, `main` bağlamına (context) aittir.
- `nginx -V` bir modülün nasıl derlendiğini gösterir: `--with-stream` gömülü,
  `--with-stream=dynamic` ise yüklenmesi gerektiği anlamına gelir.
- `http` ve `stream` ayrı üst düzey bağlamlardır; her birinin kendi `upstream` ve
  `server` blokları vardır.

Kaynak: [load_module](https://nginx.org/en/docs/ngx_core_module.html#load_module),
[ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html),
[ngx_stream_upstream_module](https://nginx.org/en/docs/stream/ngx_stream_upstream_module.html).

---

## Laboratuvar ortamı

Her şey bu dizinden oluşturulan (build) tek bir Docker konteyneri (container)
içinde çalışır:

- `Dockerfile`, imajı (image) `alpine:3.20.10` üzerine kurar; Alpine'in `nginx` ve
  `nginx-mod-stream` paketlerini, arka uç sunucular (backend) için de `python3` ve
  `socat`'ı yükler. Stream modülünün dosyası
  **/usr/lib/nginx/modules/ngx_stream_module.so**'dur.
- `start.sh` önce arka uç sunucuları, sonra NGINX'i başlatır:
  - HTTP: **127.0.0.1:8081** `l7-backend=8081`, **127.0.0.1:8082**
    `l7-backend=8082` yanıtı verir.
  - TCP: **127.0.0.1:9001** `l4-backend=9001`, **127.0.0.1:9002**
    `l4-backend=9002` yanıtı verir.
- `nginx.conf` senin dosyan. 8080 portundaki `server`'ın `location /l7` bloğu hâlâ
  yer tutucu metin döndürüyor; TODO yorumları modülü ve `stream` kısmını işaretler.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) imajı
(`nginxlings/18_osi_layers_usecases:check`) oluşturur, aktif yapılandırmayı
`nginx -T` ile okur, 8080 portunu `127.0.0.1:18180`, 9000 portunu `127.0.0.1:19080`
adresine açar, iki HTTP arka uç sunucusu da `X-L7-Upstream` başlığında görünene
kadar `/l7` ister, ardından iki TCP arka uç sunucusu da yanıt verene kadar 9000
portuna TCP bağlantıları açar.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. Stream modülünü dosyanın en üstünde yükle.
2. `http` içinde iki HTTP arka uç sunucusunu içeren bir `upstream` tanımla ve
   `location /l7` metin döndürmek yerine ona iletsin.
3. `location /l7` içinde, yanıtı veren arka uç sunucunun adresini taşıyan
   **X-L7-Upstream** yanıt başlığını ekle.
4. İki TCP arka uç sunucusunu içeren bir `upstream` ve bağlantıları ona ileten,
   **9000** portunu dinleyen bir `server` ile üst düzey bir `stream` bloğu ekle.

---

## Başarı kriterleri

- `nginx -T`, `ngx_stream_module.so` için `load_module` satırını, bir `stream`
  bloğunu ve `X-L7-Upstream` başlığını göstermeli.
- `/l7` yanıtlarında `X-L7-Upstream` içinde hem `:8081` hem `:8082` görünmeli.
- 9000 portuna yapılan TCP bağlantıları hem `l4-backend=9001`'e hem
  `l4-backend=9002`'ye ulaşmalı.

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `The stream module is not loaded yet`: başlangıç durumu bu.
- `NGINX does not know the stream directive yet`: `stream` bloğu yazılmış ama onu
  sağlayan modül yüklenmemiş.
- `"load_module" directive is specified too late`: `events` bloğundan sonra duruyor.
  `main` bağlamı nerede başlar?
- `"stream" directive is not allowed here`: `stream`, `http`'nin içine yazılmış
  (bkz. egzersiz 08).
- `No X-L7-Upstream header yet`: başlık yok ya da yorum satırında.
- `/l7 answered without an X-L7-Upstream value`: `/l7`'ye iletmeden önce hâlâ bir
  `return` yanıt veriyor ya da başlık `/l7`'ye uygulanmayan bir yerde tanımlı
  (`add_header`'ın nasıl miras alındığını hatırla, egzersiz 03).
- `Got 8081=60, 8082=0` ya da `Got 9001=40, 9002=0`: o `upstream` yalnızca bir
  arka uç sunucuya ulaşıyor. `server` satırlarını say ve portları `start.sh` ile
  karşılaştır.

---

## Pekiştirme soruları

- Her senaryo için rolü ve katmanı adlandır: NGINX TLS'i sonlandırıp HTTP uygulama
  sunucularına iletiyor; NGINX PostgreSQL bağlantılarını iki kopyaya
  dağıtıyor; NGINX arka uç yanıtlarını `proxy_cache` ile saklıyor; NGINX TLS
  bağlantılarını çözmeden SNI'ye göre farklı arka uç sunuculara gönderiyor.
- `stream` tarafında `X-L7-Upstream` başlığı yok. Bir bağlantıya hangi TCP
  arka uç sunucusunun hizmet verdiğini nerede görebilirdin?
- Egzersiz 08 `stream`'i hiç `load_module` olmadan kullandı. Orada neden çalıştı da
  burada çalışmıyor?
- Göreli yollu `load_module modules/ngx_stream_module.so;` de bu imajda çalışır.
  Neye göre göreli?
