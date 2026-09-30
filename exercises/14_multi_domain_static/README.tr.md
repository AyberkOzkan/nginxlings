# 14 – Multiple Domains, Static Sites

## Amaç

Bu egzersizde tek bir NGINX'ten, tek bir port üzerinden iki statik site sunacaksın;
NGINX siteyi her isteğin `Host` başlığına göre seçecek.

Pratik edeceklerin:

- aynı portu dinleyen birden fazla `server` bloğu,
- `server_name` ile server seçimi,
- her siteye kendi `root`'unu vermek,
- host'u hiçbir server'la eşleşmeyen bir isteğe ne olduğu.

---

## Arka plan – ada dayalı sanal sunucular

Aynı adres ve portu dinleyen birden fazla `server` bloğu olduğunda NGINX isteğin
`Host` başlığını bunların `server_name` değerleriyle karşılaştırır ve isteği eşleşen
server'a verir. Arama sırası şöyledir: tam ad, sonra `*` ile başlayan en uzun joker
(`*.example.com`), sonra `*` ile biten en uzun joker (`mail.*`), sonra eşleşen ilk
düzenli ifade.

Genel bir örnek (bu lab'dakinden farklı ad, port ve path'ler):

    server {
        listen 80;
        server_name shop.example www.shop.example;
        root /srv/shop;
    }

    server {
        listen 80;
        server_name blog.example;
        root /srv/blog;
    }

    # curl -H 'Host: blog.example' http://<adres>/  ->  /srv/blog/index.html

Bilinmesi gerekenler:

- Hiçbir `server_name` eşleşmezse istek o adres ve portun **varsayılan server**'ına
  gider: başka biri `listen ... default_server` demedikçe, o port için listelenen
  ilk server.
- Bir `server` bloğu yalnızca dinlediği portlara gelen istekleri alır. `listen`
  satırı olmayan bir blok varsayılan portu dinler (NGINX root olarak çalışıyorsa 80,
  bkz. egzersiz 03).
- Her `server`'ın kendi `root`'u olabilir; böylece aynı `/` URI'si her sitede farklı
  bir dizine eşlenir.
- `curl -H 'Host: ad' http://127.0.0.1:<port>/` ile bir adı DNS olmadan test
  edebilirsin.

Kaynak: [How nginx processes a request](https://nginx.org/en/docs/http/request_processing.html),
[Server names](https://nginx.org/en/docs/http/server_names.html).

---

## Lab ortamı

Her şey bu dizinden build edilen tek bir Docker container içinde çalışır:

- `Dockerfile`, image'i `nginx:1.28.3-alpine` üzerine kurar ve `sites/site1/`'i
  **/usr/share/nginx/site1/**'e, `sites/site2/`'yi **/usr/share/nginx/site2/**'ye
  kopyalar. Her `index.html` bir işaret içerir: `site1-static-ok` ya da
  `site2-static-ok`.
- `nginx.conf` senin dosyan. `http` bloğunda henüz server yok; TODO yorumları
  gereksinimleri listeler.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i
(`nginxlings/14_multi_domain_static:check`) build eder, aktif yapılandırmayı
`nginx -T` ile okur, container'ın 8080 portunu `http://127.0.0.1:18140/` adresine
açar ve `/`'ı iki kez ister: bir kez `Host: site1.local`, bir kez `Host: site2.local`
ile. Her gövdeyi yazdırır.

---

## Görev

`nginx.conf` dosyasını düzenle ve `http` içine iki `server` bloğu ekle:

1. İkisi de **8080** portunu dinlesin.
2. Biri **site1.local** için cevap versin ve dosyaları `/usr/share/nginx/site1`'den
   sunsun.
3. Diğeri **site2.local** için cevap versin ve dosyaları `/usr/share/nginx/site2`'den
   sunsun.
4. Her sitede `/`, o sitenin `index.html`'ini döndürsün.

---

## Başarı kriterleri

- `nginx -T`, `site1.local` ve `site2.local` için aktif `server_name` kayıtları
  göstermeli.
- `Host: site1.local` ile `GET /`, `site1-static-ok` içeren sayfayı döndürmeli.
- `Host: site2.local` ile `GET /`, `site2-static-ok` içeren sayfayı döndürmeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `No active server block with 'server_name site1.local;' yet`: başlangıç durumu bu
  ya da ad yanlış yazılmış.
- `site2.local did not return its own page` ve gövdede `site1-static-ok`: istek
  diğer server'a düşmüş. `server_name site2.local` olan bir server 8080'i dinliyor
  mu? Bir portta hiçbir ad eşleşmezse varsayılan server cevap verir.
- `site1.local did not return its own page` ve gövdede `site2-static-ok`: iki `root`
  path'i yer değiştirmiş.
- Gövde bir `404 Not Found` sayfası: `root`, o sitenin `index.html`'i olmayan bir
  dizini gösteriyor. Dockerfile'daki `COPY` satırlarıyla karşılaştır.
- `Nothing answered on port 8080`: henüz 8080'i dinleyen bir server yok.

---

## Pekiştirme soruları

- `Host: other.local` ile ya da doğrudan IP adresiyle bir istek geliyor. Hangi site
  cevap verir? Böyle isteklerin hiç cevap almamasını nasıl sağlarsın?
- Yazdıysan, `index index.html;` satırını iki server'dan da kaldır. Check yine geçer
  mi? Neden?
- Bir server'da `server_name *.site1.local;`, diğerinde
  `server_name ~^www\.site1\.local$;` var: `www.site1.local`'i hangisi alır ve
  ikisini de ne geçer?
