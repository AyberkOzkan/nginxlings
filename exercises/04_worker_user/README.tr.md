# 04 – Worker User ve İzinler

## Amaç

Bu egzersizde `user` direktifinin NGINX worker süreçlerinin hangi Unix
kullanıcısı ile çalıştığını nasıl etkilediğini ve web root dizininin sahipliğinin
neden önemli olduğunu pratik edeceksin.

Odaklandığımız noktalar:

- `user` direktifi – **main** (dosyanın en üst) context’te yer alır.
- NGINX master prosesi ile worker prosesleri arasındaki fark.
- Web root dizininin NGINX’in çalıştığı kullanıcıya göre sahipliği/izinleri.

Nginxlings içinde bu egzersizi seçip `c` veya `Enter`’a bastığında, arka planda
çalışan bir script konfigürasyonunu ve dosya izinlerini kontrol eder.

---

## Arkaplan – worker kullanıcısı

NGINX tipik bir Linux sisteminde **root** olarak başlatıldığında:

- *Master prosesi* `root` kullanıcısı olarak çalışmaya devam eder.
- Master, worker proseslerini başlatırken `user` direktifi ile bu proseslerin
  hangi kullanıcı (ve grup) ile çalışacağını belirler:

    user nginx;
    # veya
    user www-data www-data;

- Worker prosesleri istemci trafiğini bu daha az ayrıcalıklı kullanıcı ile
  işler.

Notlar:

- `user` direktifi sadece **main** context’te (nginx.conf’un en üst seviyesi)
  geçerlidir; `http`, `server` veya `location` içinde yazılmaz.
- Tipik kullanıcılar `nginx`, `www-data` veya uygulamaya özel oluşturulmuş
  bir servistir.
- Web root dizinindeki dosyaların sahipliği ve izinleri, bu kullanıcının
  dosyaları okuyup (gerekirse yazıp) yazamayacağını belirler.

Bu lab ortamında NGINX’i normal bir kullanıcı ile çalıştırdığında, `user`
direktifi bazı durumlarda NGINX tarafından göz ardı edilebilir; yine de aynı
sözdizimini kullanırız ve pratik yapmak için faydalıdır.

---

## Bu egzersizde ne yapıyorsun?

Sadece bu dizin içinde çalışıyorsun:

    04_worker_user/

Genel fikir:

1. NGINX’i şöyle ayarlamak:
   - **Main** context’te, Nginxlings’i çalıştıran Unix kullanıcısıyla aynı
     kullanıcıyı kullanan bir `user` direktifi olmalı.
   - Bu klasör içinde yer alan `html/` dizininden dosya sunan basit bir HTTP
     server olmalı.
2. `html/` dizininin ve içindeki `index.html` dosyasının **aynı kullanıcıya
   ait** olduğundan emin olmak.

Denetleyici script, geçerli Unix kullanıcısını ve grubunu şu komutlarla
belirler:

    id -un
    id -gn

ve `user` direktifinin bu kullanıcıya uygun olmasını bekler.

---

## Hazırlaman gerekenler

`04_worker_user` dizininde:

1. **Ana konfigürasyon dosyası**

    Bu dizinde `nginx.conf` dosyasını oluştur ya da düzenle. Minimum olarak:

    - **Main** context’te, geçerli kullanıcıyı kullanan bir `user` direktifi
      olmalı (terminalde `whoami` çıktısı ile aynı):

        user seninkullanici;
        # veya grup ile birlikte:
        user seninkullanici sengrubun;

      Denetleyici şu formları kabul eder:

      - `user <gecerli_kullanici>;`
      - `user <gecerli_kullanici> <gecerli_grup>;`

    - Basit bir `events` bloğu:

        events {
            worker_connections 1024;
        }

    - Basit bir `http` bloğu ve server, örneğin:

        http {
            server {
                listen 8082;
                server_name localhost;

                # Web root, bu egzersiz klasörünün altındaki html/
                root html;
                index index.html;

                location / {
                }
            }
        }

    Loglama, gzip vb. ekleyebilirsin; önemli olan sözdiziminin geçerli olması
    ve `nginx -t` testinin hatasız geçmesidir.

2. **Web root dizini ve dosya**

    Bu klasörde bir `html/` dizini ve içinde bir `index.html` dosyası oluştur:

        html/
          index.html

    `index.html` içeriği sana kalmış; denetleyici sadece dosyanın varlığına ve
    sahipliğinin doğru olmasına bakar.

3. **Dosya sahipliği**

    Şunlardan emin ol:

    - `html/` dizini, bu egzersizi çalıştıran kullanıcıya ait.
    - `html/index.html` dosyası da aynı kullanıcıya ait.

    Sahipliği şu şekilde görebilirsin:

        ls -ld html
        ls -l html/index.html

    Gerekirse `chown` ile düzelt:

        chown kullanici:grup html html/index.html

    (Bazı sistemlerde bunun için `sudo` gerekebilir.)

---

## Denetleyicinin kontrol ettiği noktalar

Nginxlings içinde `c` veya `Enter`’a bastığında, bu dizindeki `check.sh`
script’i çalışır ve şu kontrolleri yapar:

1. **Konfigürasyon sözdizimi**

    - Bu dizinde bir `nginx.conf` dosyası olmalı.
    - `nginx -t -c nginx.conf -p .` komutu hatasız bitmeli.

2. **`user` direktifi**

    - `nginx.conf` içinde, main context’te geçerli Unix kullanıcısına uyan
      bir `user` direktifi bulunmalı.
    - Kabul edilen formlar:

        user <gecerli_kullanici>;
        user <gecerli_kullanici> <gecerli_grup>;

3. **Web root ve sahiplik**

    - Bu klasörde bir `html/` dizini olmalı.
    - `html/index.html` dosyası bulunmalı.
    - Hem `html/` dizini hem de `html/index.html` dosyası geçerli kullanıcıya
      ait olmalı.

Tüm bu kontroller geçtiğinde, Nginxlings arayüzünde bu egzersiz tamamlandı (✓)
olarak işaretlenir.

---

## İpuçları

- `user` direktifi mutlaka **nginx.conf’un en üst seviyesinde** olmalı; `http`
  bloğunun içine yazma.
- Denetleyici sahiplikle ilgili uyarı veriyorsa, `ls -l` ile kimin sahibi
  olduğuna bak ve gerekirse `chown` ile düzelt.
- NGINX’i normal kullanıcı ile çalıştırdığın ortamlarda worker prosesleri
  zaten o kullanıcı ile çalışıyor olabilir; burada amaç `user` direktifine
  ve worker kullanıcısı–dosya izinleri ilişkisine alışmak.

