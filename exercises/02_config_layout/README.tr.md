# 02 – Config Layout ve Varsayılan Dizinler

## Amaç

Bu egzersizde NGINX yapılandırma dosyalarının tipik yerleşimini
ve `nginx -t` / `nginx -T` çıktısını okuyarak yapılandırmayı nasıl inceleyeceğini öğreneceksin.

Gerçek bir Linux sisteminde genelde:

- Ana yapılandırma dosyası: `/etc/nginx/nginx.conf`
- Ek sanal host / server tanımları: `/etc/nginx/conf.d/*.conf` veya `sites-enabled/`

Bu lab’de aynı mantığı **bu klasör içinde** küçük ölçekte simüle ediyoruz:

- Bu dizindeki `nginx.conf` bizim “ana” config dosyamız.
- `conf.d/` altındaki dosyalar `include` direktifi ile yükleniyor.

Nginxlings arayüzünden bu egzersizi seçip `c` veya Enter’a bastığında
yapılandırmayı kontrol eden script çalışır.

---

## Konu Özeti

Bu egzersiz şunları pekiştirmek için var:

- Varsayılan NGINX dizin mantığı (`/etc/nginx/` benzeri yapı)
- Ana config dosyası: `nginx.conf`
- Ek config dosyaları: `conf.d/*.conf`
- `include` direktifinin çalışma mantığı
- `nginx -t` ve `nginx -T` ile yapılandırma inceleme

Gerçek dünyada:

- `nginx -t` → yapılandırma dosyasının sözdizimini ve temel geçerliliğini kontrol eder.
- `nginx -T` → include edilmiş dosyalarla birlikte **etkin** yapılandırmanın tamamını yazar.

Bu lab’de bu komutları kendi local config’imiz üzerinde deniyorsun.

---

## Yapman Gerekenler

Bu egzersiz üzerinde çalışırken `02_config_layout` klasöründe ol:

1. Ana config dosyasını incele (`nginx.conf`)

   - Bu dizindeki `nginx.conf` dosyasını aç.
   - Dosyada en azından şu temel bloklar olmalı:

        events { ... }
        http   { ... }

   - `http { ... }` bloğunun içinde aşağıdaki gibi **aktif** bir include satırı olduğundan emin ol:

        include conf.d/*.conf;

     (Başında `#` olmamalı. Bu satır, `conf.d/` altındaki `.conf` dosyalarını yükler.)

2. conf.d dizinini ve bir sanal host dosyasını oluştur

   - `conf.d/` dizini yoksa oluştur:

        mkdir -p conf.d

   - `conf.d/default.conf` adında bir dosya oluştur ve içine basit bir `server` bloğu yaz.
     Örnek:

        server {
            listen 8081;
            server_name localhost;

            root .;

            location / {
                return 200 "config-layout-ok\n";
            }
        }

   - Buradaki amaç:
     - `server` bloğunun ana dosyada değil, `conf.d/default.conf` içinde olması.
     - Ana dosya (`nginx.conf`) sadece genel ayarlar + `include conf.d/*.conf;` satırını içersin.

3. Yapılandırmayı test et (isteğe bağlı, öğrenme için)

   Bu klasördeyken şu komutları çalıştırabilirsin:

        nginx -t -c nginx.conf -p .

   - `-c nginx.conf` → hangi config dosyasını kullanacağını söyler.
   - `-p .`        → NGINX için “prefix” olarak bu dizini kullan (log, pid vb.).

   Etkin config’i görmek için:

        nginx -T -c nginx.conf -p .

   Çıktı içinde hem `include conf.d/*.conf;` satırını
   hem de `conf.d/default.conf` içeriğinin gömülü hâlini görmelisin.

4. Egzersizi Nginxlings üzerinden doğrula

   - Nginxlings arayüzünde bu egzersizi seç.
   - `c` veya Enter’a bas.
   - Arka planda çalışan `check.sh` script’i:
     - `nginx -t` ile config’i test eder,
     - `nginx -T` çıktısında `include conf.d/*.conf;` satırını arar.

   Her şey yolundaysa egzersiz “tamamlandı” (✓) olarak işaretlenir.

---

## Başarı Kriterleri

Egzersiz **başarılı** sayılması için:

- `nginx -t -c nginx.conf -p .` komutu hatasız bitmeli (exit code 0).
- `nginx -T -c nginx.conf -p .` çıktısında
  `include conf.d/*.conf;` satırı bulunmalı.
- `conf.d/default.conf` içinde en az bir `server { ... }` bloğu olmalı
  ve bir portta (örneğin 8081) dinliyor olmalı.

---

## İpuçları

- Gerçek bir sistemde ana yapılandırma dosyası genellikle:

      /etc/nginx/nginx.conf

- Ek sanal host’lar için sık kullanılan dizin:

      /etc/nginx/conf.d/

  Bu lab’deki `nginx.conf` ve `conf.d/` yapısını,
  bu gerçek dizinlerin küçük bir kopyası gibi düşünebilirsin.

- `nginx -T` çıktısı; sorun giderirken, konfigürasyonu baştan sona hızlıca görmek için
  oldukça kullanışlıdır. Büyük `T` harfine dikkat et.

