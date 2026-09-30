# 02 – Yapılandırma Düzeni ve Varsayılan Yollar

## Amaç

Bu egzersizde bir NGINX yapılandırmasının genellikle bir ana dosya ile ek
dosyalar arasında nasıl bölündüğünü, NGINX'in gerçekte hangi dosyaları yüklediğini
nasıl öğreneceğini ve bir yapılandırmayı `nginx -t` ve `nginx -T` ile nasıl test
edeceğini öğreneceksin.

---

## Arka plan – yapılandırma nerede durur

- **Ana dosya** (çoğu Linux sistemde `/etc/nginx/nginx.conf`), ana bağlamı (main
  context; `user`, `worker_processes`, `error_log`, `pid`), `events {}` bloğunu ve
  her site için ortak ayarların bulunduğu `http {}` bloğunu tutar.
- **Sunucu tanımları** genellikle ana dosyanın `http {}` içinde `include` ile
  yüklediği ayrı dosyalarda durur. İki yaygın düzen:
  - `/etc/nginx/conf.d/*.conf`: eşleşen her dosya yüklenir (nginx.org paketleri ve
    resmi Docker imajı (image)).
  - `sites-available/` + `sites-enabled/`: bütün siteler ilk dizinde tutulur,
    yalnızca ikinciye bağlantısı verilenler yüklenir (Debian ve Ubuntu paketleri).
- `nginx -V`, bir kurulumun ana dosyasını nerede beklediğini gösterir
  (`--conf-path=`). `-c <dosya>` başka bir dosya seçer, `-p <dizin>` ise
  `logs/error.log` gibi göreli yolların çözüleceği öneki (prefix) belirler. Göreli bir
  `include` ana dosyanın bulunduğu dizine göre çözülür.
- `nginx -t` yapılandırmayı test eder. `nginx -T` test eder ve NGINX'in yüklediği
  her şeyi, her dosyanın önüne bir `# configuration file ...:` başlığı koyarak
  yazdırır.

Genel bir örnek (bu laboratuvardakinden farklı dizin ve adlar):

    http {
        include /etc/nginx/vhosts/*.conf;
    }

    # /etc/nginx/vhosts/blog.conf
    server {
        listen 8000;
        server_name blog.local;
        ...
    }

Kaynak: [Beginner's Guide](https://nginx.org/en/docs/beginners_guide.html),
[Command-line parameters](https://nginx.org/en/docs/switches.html).

---

## Laboratuvar ortamı

Bu egzersiz NGINX'i Docker'da değil, doğrudan makinende çalıştırır ve `/etc/nginx`
düzenini bu dizinin içinde taklit eder:

- `nginx.conf` ana dosyadır. `http {}` bloğunda ek dosyaların yükleneceği yerde bir
  `TODO` var.
- `conf.d/default.conf` ek dosyadır. Yalnızca bir `TODO` yorumu içerir.

Çalışmanı bu dizinden kendin de test edebilirsin:

    nginx -t -c nginx.conf -p .
    nginx -T -c nginx.conf -p .

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) bu dizini önek alarak
`nginx -t` çalıştırır, `nginx -T` çıktısını okur (yorum satırları dikkate alınmaz),
`include`'un etkin olduğunu ve `conf.d/default.conf`'un etkin bir `server` bloğuyla
yüklendiğini doğrular, bu bloğun `listen` portunu okur, NGINX'i başlatır ve
`http://127.0.0.1:<port>/` ister.

---

## Görev

1. `nginx.conf` içinde, `http {}` içinde, `conf.d/` dizinindeki her `.conf`
   dosyasını yükle.
2. `conf.d/default.conf` içinde, senin seçtiğin boş bir portu (örneğin 8081)
   dinleyen, sunucu adı olarak `localhost` kullanan ve her isteğe sonunda satır
   sonu olan `config-layout-ok` ile yanıt veren bir `server` bloğu tanımla.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- `nginx -T`, `conf.d/*.conf` için etkin bir `include`'u ve etkin bir `server` bloğu
  ile `listen` portu içeren `conf.d/default.conf` içeriğini göstermeli.
- Bu porta yapılan istek HTTP 200 ile `config-layout-ok` dönmeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `nginx.conf does not load the .conf files in conf.d/ yet`: `include` eksik, hâlâ
  yorum satırı ya da başka bir yeri gösteriyor.
- `NGINX rejected the configuration: "server" directive is not allowed here in
  .../conf.d/default.conf`: `include` `http {}` dışında. `default.conf`'a değil,
  `include` satırına bak.
- `No active 'server { ... }' block found in conf.d/default.conf`: dosya
  yüklenmiyor ya da `server` bloğu hâlâ yorum satırı.
- `No active 'listen <port>;' directive found`: `server` bloğunun bir porta
  ihtiyacı var.
- `Port ... is already in use by ...`: o portu başka bir program kullanıyor. Başka
  boş bir port seç.
- `The response ... does not contain 'config-layout-ok'`: `location /`'un ne yanıt
  verdiğine bak.

---

## Pekiştirme soruları

- Makinende kurulu NGINX'in ana yapılandırma dosyası nerede ve NGINX bunu sana
  kendisi nasıl söyleyebilir?
- `conf.d/*.conf` düzeni ile `sites-available/` + `sites-enabled/` düzeni arasındaki
  fark nedir? Senin makinen hangisini kullanıyor?
- Bu laboratuvarda `-c` ve `-p` neyi değiştirir ve `logs/error.log` hangi dosyaya
  yazılır?
- `nginx -T` çıktısında hangi dosyaların dahil edildiğini (`include`) nasıl anlarsın?
