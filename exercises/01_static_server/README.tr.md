# 01 – Basic Static NGINX Server

## Amaç

Bu egzersizde NGINX'i bir **statik web sunucusu** olarak çalıştıracaksın: `/`
isteğine diskten okunan bir dosyayla cevap verilecek.

Pratik edeceklerin:

- bir istek URI'sini `root` ile bir dosyaya eşlemek,
- bir dizin için varsayılan dosyayı `index` ile seçmek,
- göreli path'lerin NGINX **prefix**'ine göre nasıl çözüldüğü.

---

## Arka plan – URI'den dosyaya

Statik bir istekte NGINX, istek URI'sini `root` dizininin sonuna ekler ve o
dosyayı sunar. URI `/` ile bitiyorsa bir dizini gösterir; bu durumda `index`
direktifi o dizinde sırayla denenecek dosya adlarını listeler ve var olan ilk dosya
sunulur.

Genel bir örnek (bu lab'dakinden farklı ad ve portlar):

    server {
        listen 9090;
        root /srv/docs;
        index start.html;
    }

    # GET /          -> /srv/docs/start.html
    # GET /css/a.css -> /srv/docs/css/a.css

Bilinmesi gerekenler:

- `root`, `error_log`, `pid` veya `access_log` içindeki göreli bir path, bulunduğun
  dizine ya da `nginx.conf`'un dizinine göre değil, **prefix**'e göre çözülür.
  Prefix `nginx -p <dizin>` ile verilir; verilmezse NGINX derlendiği prefix'i
  kullanır (`nginx -V` bunu gösterir).
- Dizin var ama `index` dosyalarının hiçbiri yoksa NGINX `403` döner (`autoindex on;`
  yoksa dizinleri listelemez). Path hiç yoksa `404` döner.
- Error log, NGINX'in denediği path'i tam olarak yazar; göreli bir `root`'un
  gerçekte nereyi gösterdiğini buradan görürsün.
- `return 200 "metin";` hiçbir dosya okumadan cevap verir. Bu egzersizin konusu
  dosyayı sunmaktır.

Kaynak: [root](https://nginx.org/en/docs/http/ngx_http_core_module.html#root),
[index](https://nginx.org/en/docs/http/ngx_http_index_module.html),
[Command-line parameters](https://nginx.org/en/docs/switches.html).

---

## Lab ortamı

Bu egzersiz makinende kurulu NGINX'i kullanır:

- `nginx.conf` senin dosyan. **8080** portunu dinler ve `root` ile `index` için hâlâ
  yer tutucular içerir. Error log'u bu dizindeki `logs/error.log` dosyasına gider.
- `index.html` sunulacak sayfa. Hâlâ bir yer tutucu içeriyor.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`)
`nginx -t -c nginx.conf -p <bu dizin>` çalıştırır, NGINX'i aynı prefix ile
foreground'da başlatır, `http://127.0.0.1:8080/` ister, cevabı `index.html` ile
byte byte karşılaştırır ve NGINX'i yeniden durdurur. 8080 portu boş olmalı.

---

## Görev

1. `nginx.conf` içindeki iki yer tutucuyu, `/` isteğine bu egzersiz dizinindeki
   `index.html` ile cevap verilecek şekilde değiştir. `listen 8080;` satırını koru.
2. `index.html` içindeki yer tutucuyu, sayfa tam olarak `Merhaba Nginxlings`
   metnini içerecek şekilde değiştir.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- `http://127.0.0.1:8080/` `200` dönmeli.
- Cevap, bu dizindeki `index.html`'in içeriğiyle byte byte aynı olmalı.
- `index.html`, `Merhaba Nginxlings` içermeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `got 404`: NGINX dizini ya da dosyayı bulamadı. `logs/error.log` dosyasını aç:
  `is not found` satırı denenen path'in tamamını gösterir. Prefix hangi dizini
  gösteriyor ve göreli `root`'un buna ne ekliyor?
- `got 403`: dizin bulundu ama `index`'teki adların hiçbiri orada yok (error log'da
  `directory index of "..." is forbidden`).
- `index.html does not contain 'Merhaba Nginxlings' yet`: NGINX çalışıyor; şimdi
  sayfayı düzenle.
- `The response to / is not the content of index.html`: bir `return`, dosya yerine
  metinle cevap veriyor.
- `bind() to 0.0.0.0:80 failed (13: Permission denied)`: 1024 altındaki portlar
  root ister. 8080 portunu koru.
- `Port 8080 ... is already in use by ...`: portu başka bir program tutuyor. Onu
  durdurup check'i yeniden çalıştır.

---

## Pekiştirme soruları

- `nginx -V` bir `--prefix=` gösterir. NGINX `-p` olmadan başlatılsaydı
  `root html;` nereyi gösterirdi?
- `index a.html b.html index.html;` yazılıysa ve diskte `a.html` yoksa `/` için
  hangi dosya sunulur?
- `autoindex on;` eklersen ve `index` var olmayan bir dosyayı gösterirse `GET /` için
  ne değişir?
