# 01 – Basit Statik NGINX Sunucusu

## Amaç

Bu egzersizde NGINX’i basit bir **statik web sunucusu** olarak çalıştıracaksın.

Nginxlings içinden bu egzersizi seçip `c` veya Enter’a bastığında yapılan kontrolde,
`http://127.0.0.1:8080/` adresinden dönen sayfanın gövdesinde şu metin bulunmalıdır:

> Merhaba Nginxlings

---

## Yapman Gerekenler

1. **`nginx.conf` dosyasını düzenle**

   Aşağıdaki gereksinimleri karşılayacak şekilde yapılandır:

   - Bir `http { ... }` bloğu içinde **en az bir `server { ... }`** tanımı olmalı.
   - Bu sunucu **8080 portunu** dinlemeli:
     ```nginx
     listen 8080;
     ```
   - `root` dizini bu egzersiz klasörü olacak şekilde ayarlanmalı.
     (Nginxlings, `check.sh` içinde bu klasörü NGINX için prefix olarak kullanır.)
   - Varsayılan index dosyası `index.html` olmalı:
     ```nginx
     index index.html;
     ```

2. **`index.html` dosyasını düzenle**

   - Geçerli bir HTML sayfası oluştur.
   - `<body>` içinde **“Merhaba Nginxlings”** metni mutlaka geçsin.
     (Denetleyici bu metni arar.)

3. **Egzersizi Nginxlings içinden kontrol et**

   - Nginxlings arayüzünde bu egzersizi seç.
   - `c` veya Enter’a bas.
   - Arka planda `check.sh` çalışır:
     - `nginx -t` ile konfigürasyonunu test eder,
     - 8080 portunda NGINX’i başlatır,
     - `curl` ile sayfayı ister ve içerikte “Merhaba Nginxlings” metnini arar,
     - İş bittikten sonra NGINX’i durdurur.

---

## Başarı Kriterleri

Egzersiz **başarılı** sayılması için:

- NGINX yapılandırması `nginx -t` ile hatasız olmalı,
- `http://127.0.0.1:8080/` isteği **200 OK** döndürmeli,
- Cevap gövdesinde **“Merhaba Nginxlings”** metni bulunmalı.

Bu şartlar sağlandığında Nginxlings arayüzünde bu egzersiz **tamamlandı (✓)** olarak işaretlenir.

---

## İpuçları

- Özel bir konfigürasyon dosyası ile NGINX çalıştırma mantığı:
  ```nginx
  nginx -c /yol/nginx.conf -p /yol/
  ```

- “Bu egzersizde bu komutları elle yazmana gerek yok; …”
- “8080 portu kullanıldığı için root yetkisi gerekmez.”
