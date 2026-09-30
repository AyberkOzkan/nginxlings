# 03 – Context Hierarchy (main / http / server / location)

## Amaç

Bu egzersizde direktifleri yanlış context'lerde duran bir `nginx.conf`'u, her birini
`nginx -t` mesajlarıyla bularak onaracaksın.

Pratik edeceklerin:

- `main` > `http` > `server` > `location` iç içe yapısı,
- `"..." directive is not allowed here` mesajını ve gösterdiği satır numarasını
  okumak,
- bir direktifin hangi context'e ait olduğuna karar vermek.

---

## Arka plan – context'ler

Bir NGINX yapılandırması, **context** denen blokların oluşturduğu bir ağaçtır. Her
direktif yalnızca belirli context'lerde geçerlidir ve bir blok direktifi yeni bir
context açar:

    # main: NGINX'in tamamı için ayarlar
    worker_processes 2;

    events {
        # bağlantı işleme
    }

    http {
        # tüm HTTP trafiği için ayarlar
        keepalive_timeout 30;

        server {
            # tek bir sanal sunucu: adres, port, adlar
            listen 9090;
            server_name docs.example;

            location /api/ {
                # URI'si /api/ ile başlayan istekler
            }
        }
    }

Bilinmesi gerekenler:

- Her direktifin dokümantasyonunda nerede kullanılabileceğini listeleyen bir
  **Context:** satırı vardır. `nginx -t` buna uyar ve ilk hatalı satırı gösterir:
  `"listen" directive is not allowed here in .../nginx.conf:31`.
- `http` ve `events` `main`'e, `server` `http`'ye, `location` ise `server`'a (ya da
  başka bir `location`'a) aittir.
- Dıştaki bir context'teki ayar, içtekiler kendileri ayarlamadıkça onlara da geçer:
  `server` içindeki bir `root`, o server'ın location'larına da uygulanır.
- `nginx -t` ilk hatada durur. Her düzeltmeden sonra yanlış yerdeki bir sonraki
  direktif ortaya çıkar.

Kaynak: [Beginner's guide – configuration file structure](https://nginx.org/en/docs/beginners_guide.html#conf_structure),
[ngx_http_core_module](https://nginx.org/en/docs/http/ngx_http_core_module.html).

---

## Lab ortamı

Bu egzersiz makinende kurulu NGINX'i kullanır:

- `nginx.conf` senin dosyan. Üst kısmı (`main`, `events`, `http`) geçerli;
  `INTENTIONALLY BROKEN CONTEXT SECTION` ile işaretli bölüm ise değil.
- `logs/error.log` gibi göreli path'ler bu dizine göre çözülür.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`)
`nginx -t -c nginx.conf -p <bu dizin>` çalıştırır, NGINX'i foreground'da başlatır,
`http://127.0.0.1:8080/` ister ve gövdesinde `context-hierarchy-ok` bulunan bir
`200` bekler. 8080 portu boş olmalı.

---

## Görev

`nginx.conf` dosyasını şöyle düzenle:

1. `listen 8080;` ve `server_name localhost;` `server` bloğuna ait olsun.
2. `location /` ile birlikte `server` bloğu `http`'nin içinde yer alsın.
3. `location /` `return`'ünü korusun; `/` yine `context-hierarchy-ok` dönsün.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- `http://127.0.0.1:8080/` `200` dönmeli.
- Gövde `context-hierarchy-ok` içermeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `"listen" directive is not allowed here in ...:31`: başlangıç durumu bu. `listen`
  hangi bloğu tanımlar?
- `"server" directive is not allowed here`: `server` bloğu hâlâ `http`'nin dışında.
- `"location" directive is not allowed here`: bir `location` doğrudan `http`
  içinde. Onu hangi blok sarmalı?
- `"sendfile" directive is not allowed here` veya
  `"worker_processes" directive is not allowed here`: bir direktif desteklemediği
  bir context'e taşınmış. Dokümantasyondaki **Context:** satırına bak.
- `NGINX did not answer on http://127.0.0.1:8080/`: yapılandırma geçerli ama 8080'i
  dinleyen bir `server` yok. `listen 8080;` taşırken kayboldu mu?
- `got 403`: `location /` içindeki `return` gitmiş; NGINX `root`'tan dosya sunmaya
  çalışıyor ve bu dizinde `index.html` yok.

---

## Pekiştirme soruları

- Hiç `listen` direktifi olmayan bir `server` yine bağlantı kabul eder. Hangi
  portta, ve cevap neden NGINX'in root olarak başlayıp başlamamasına bağlı?
- `sendfile` `http`, `server` ve `location` içinde kullanılabilir. `http`'de
  `sendfile on;`, bir location'da `sendfile off;` yazıyorsa diğer location'lara
  hangisi uygulanır?
- Bir `server`'da `add_header X-A ...;`, location'larından birinde
  `add_header X-B ...;` var. O location'a gelen istek hangi başlıkları alır, başka
  bir location'a gelen istek hangilerini alır?
