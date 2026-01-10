# 03 – Context Hierarchy (main / http / server / location)

## Amaç

Bu egzersizde, NGINX konfigürasyon dosyasındaki **context hiyerarşisini** anlamayı
ve hatalı yerleştirilmiş direktifleri doğru seviyeye taşımayı öğreneceksin.

Özellikle şu zinciri pekiştiriyoruz:

- `main` (dosyanın en üst seviyesi)
- `http`
- `server`
- `location`

Egzersizde bazı direktifler **bilerek yanlış context içine** yazılmış durumda olacak.
Görev: Otomatik kontrol sırasında görünen hata/ipuçlarını kullanarak,
bu direktifleri doğru context hiyerarşisine yerleştirmek.

---

## Arkaplan – Context hiyerarşisi

NGINX konfigürasyonunda context, blokların içine yazdığın direktiflerin
hangi scope içinde geçerli olduğunu belirler.

- **main context**  
  Dosyanın en üst seviyesi. Genelde şunlar burada olur:

    user ...
    worker_processes ...
    error_log ...
    pid ...


- **http context**  

    http {
        # HTTP ile ilgili genel ayarlar
        sendfile on;
        keepalive_timeout 65;

        server {
            ...
        }
    }


- **server context**  

    server {
        listen 8080;
        server_name localhost;
        root /var/www/html;

        location / {
            ...
        }
    }


- **location context**  

    location /images/ {
        # URI eşleşmesine göre davranış
    }

Bazı direktifler sadece belirli context’lerde kullanılabilir. Örneğin:

- `user`, `worker_processes` → sadece main seviyesinde anlamlıdır.
- `sendfile`, `keepalive_timeout` → HTTP seviyesinde kullanılır.
- `listen`, `server_name`, `root` → genellikle `server` içinde görülür.
- URI eşleşmesine bağlı detaylar → `location` içinde tanımlanır.

Bu egzersizde, bilerek yanlış yere konmuş direktifler olacak ve bunlar
otomatik kontrol sırasında hata/uyarı üretecek.

---

## Bu egzersizde ne yapıyorsun?

Bu dizindeki `nginx.conf` dosyası:

- Temel context hiyerarşisini içerir,
- Ancak bazı direktifler kasıtlı olarak yanlış context’e yerleştirilmiştir.

Senin görevin:

1. `c` veya Enter ile kontrolü çalıştırmak.
2. Status satırında ve gerekiyorsa detaylarda gösterilen ipuçlarına bakmak:
   - Örneğin “directive X is not allowed here” gibi mesajlar,
   - Hangi direktifin yanlış context’te olduğunu ima eder.
3. `nginx.conf` içindeki blok yapısını ve direktiflerin konumlarını,
   context hiyerarşisini bozmadan mantıklı hale getirmek.

Burada önemli olan:

- `http` bloğunun **main** seviyesinde olması,
- `server` bloklarının **http** içinde yer alması,
- `location` bloklarının **server** içinde yer alması,
- Context’e uygun olmayan direktiflerin bulunduğu seviyeden alınarak
  mantıksal olarak ait oldukları seviyeye taşınmasıdır.

---

## Başarı kriterleri

Bu egzersiz **başarılı** sayılması için:


- Yapılandırma syntax açısından geçerli olmalı,
- Context hiyerarşisi bozuk olmamalı,
- Hatalı context’e yerleştirilmiş direktifler düzeltilmiş olmalı.
- `server` ve `location` blokları doğru context içinde yer almalı.
- Context ile ilgili hata mesajları (örneğin “is not allowed here”) artık görünmemeli.
- Egzersiz listesinde bu adım ✓ (tamamlandı) olarak işaretlenmeli.

---

## İpuçları

- Hata/uyarı mesajlarında genellikle hangi direktifin, hangi satırda
  yanlış context’te olduğu açıkça belirtilir.
- Bir direktifin nerede durması gerektiğinden emin değilsen:
  - Önce mantıksal olarak düşün: Bu ayar tüm HTTP trafiğini mi etkiliyor,
    sadece belli bir sanal sunucuyu mu, yoksa belli bir path’i mi?
  - Genel kural:  
    - Tüm NGINX için → main  
    - HTTP katmanı için → http  
    - Belirli host/port kombinasyonu için → server  
    - URI/path özel davranışı için → location
- Aynı tipte bir direktifi (örneğin `listen`) farklı context’lerde görüyorsan,
  büyük ihtimalle yanlış yerde olanı vardır.

