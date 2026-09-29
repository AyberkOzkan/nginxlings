# 13 – Include & Modular Config Design

## Amaç

Bu egzersizde bir NGINX yapılandırmasını `include` ile birden fazla dosyaya
bölecek ve `include`'un gerçekte ne yaptığını göreceksin: include edilen metin,
**`include` satırında yazılmış gibi**, o satırın context'inde işlenir.

---

## Arka plan – include nasıl çalışır

- `include <yol>;` dosyanın (ya da `*.conf` gibi bir glob'a uyan her dosyanın)
  içeriğini o noktaya ekler. Include edilen dosyada NGINX'e context'ini söyleyen
  hiçbir şey yoktur: `server {}` blokları içeren bir dosya yalnızca `server`'ın
  geçerli olduğu yerde (`http {}` içinde), `location {}` blokları içeren bir dosya
  yalnızca `server {}` içinde çalışır.
- Bir glob'a uyan dosyalar alfabetik sırayla yüklenir; modüler yapılarda dosya
  adlarının sık sık sayıyla başlamasının sebebi budur.
- Göreli bir yol yapılandırma prefix'ine göre çözülür (bu lab'ın image'ında
  `/etc/nginx`).
- `nginx -T` yapılandırmayı test eder ve etkin sonucun tamamını, yüklenen her
  dosyanın önüne bir `# configuration file ...:` başlığı koyarak yazdırır. Bir
  `include`'un gerçekte neyi çektiğini görmenin en hızlı yolu budur.

Genel bir örnek (bu lab'dakinden farklı dizin ve adlar):

    http {
        include /etc/nginx/sites/*.conf;          # her dosyada bir server {}
    }

    # /etc/nginx/sites/shop.conf
    server {
        listen 80;
        server_name shop.example;
        include /etc/nginx/snippets/headers.conf; # add_header satırları
    }

Kaynak: [include](https://nginx.org/en/docs/ngx_core_module.html#include).

---

## Lab ortamı

- `Dockerfile`, bir `nginx:1.28.3-alpine` image'ında `nginx.conf`'u
  `/etc/nginx/nginx.conf`'a, `conf.d/` dosyalarını `/etc/nginx/conf.d/`'ye,
  `server.d/` dosyalarını `/etc/nginx/server.d/`'ye kopyalar. Bu image'da zaten
  kendi `/etc/nginx/conf.d/default.conf` dosyası var.
- `conf.d/10-server.conf`, 8080 portunda `location /`'u `base-ok` cevabı veren bir
  `server` içerir.
- `server.d/20-modular.conf` ve `server.d/30-modular-extra.conf` yalnızca TODO
  yorumları içerir.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i build eder, etkin
yapılandırmayı `nginx -T` ile okur (yorum satırları dikkate alınmaz), iki dizinin
de include edildiğini doğrular, container'ı başlatır (host portu 18130) ve `/`,
`/modular` ile `/modular2` ister. Her gövde birebir eşleşmeli; sondaki satır sonu
dikkate alınmaz.

---

## Görev

1. `nginx.conf` içinde, `http {}` içinde, `/etc/nginx/conf.d/` altındaki her `.conf`
   dosyasını yükle.
2. `conf.d/10-server.conf` içinde, `server {}` içinde, `/etc/nginx/server.d/`
   altındaki her `.conf` dosyasını yükle.
3. `server.d/20-modular.conf` içinde `/modular`, sonunda satır sonu olan
   `include-ok` ile cevap versin.
4. `server.d/30-modular-extra.conf` içinde `/modular2`, sonunda satır sonu olan
   `include2-ok` ile cevap versin.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- Etkin yapılandırma `/etc/nginx/conf.d/*.conf` ve `/etc/nginx/server.d/*.conf`
  dosyalarını include etmeli.
- `GET /` `base-ok`, `GET /modular` `include-ok`, `GET /modular2` `include2-ok`
  dönmeli; hepsi HTTP 200 ile.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `nginx.conf does not load the .conf files in /etc/nginx/conf.d/ yet`: ilk include
  eksik ya da hâlâ yorum satırı.
- `NGINX rejected the configuration: "server" directive is not allowed here in
  /etc/nginx/conf.d/10-server.conf`: `conf.d` include'u `http {}` dışında. Hata
  include edilen dosyayı gösterir ama düzeltilecek yer `include` satırıdır.
- `"location" directive is not allowed here in /etc/nginx/server.d/...`:
  `server.d` include'u `server {}` yerine `http {}` içinde.
- `Expected GET /modular to return ... Is location /modular defined`: isteğe
  `location /` cevap vermiş (`base-ok`); yani `/modular` için hiçbir location
  yüklenmemiş.
- `duplicate location`: iki include dosyası aynı location'ı tanımlıyor.

---

## Pekiştirme soruları

- `server.d/*.conf`'a uyan dosyalar hangi sırayla yüklenir ve birçok yapıda dosya
  adları neden sayıyla başlar?
- `conf.d` include'un senin yazmadığın bir `server` bloğunu daha yüklüyor.
  Hangisi ve neden seninkiyle çakışmıyor?
- NGINX'in yüklediği yapılandırmanın tamamını dosya dosya nasıl görürsün?
- `20-modular.conf` ve `30-modular-extra.conf` ikisi de `/modular` için bir
  location tanımlarsa ne olur?
