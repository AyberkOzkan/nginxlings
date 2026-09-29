# 06 – HTTP Reverse Proxy Temelleri

## Amaç

Bu egzersizde NGINX'i küçük bir backend uygulamasının önüne **HTTP reverse
proxy** olarak koyacaksın: istemciler yalnızca NGINX ile konuşur, cevabı
backend'den NGINX getirir.

---

## Arka plan – reverse proxy ne yapar

Reverse proxy, istemcinin bağlantısını kabul eder, backend sunucuya kendi
bağlantısını açar, isteği iletir ve backend'in cevabını geri döndürür. İstemci
backend'in adresini hiç görmez.

NGINX'te bu iş bir `location` içindeki `proxy_pass`'e düşer. `proxy_pass`, şemalı
(`http://` veya `https://`) bir URL ve bir adres alır. Genel bir örnek (bu
lab'dakinden farklı ad ve adresler):

    server {
        listen 80;

        location /shop/ {
            proxy_pass http://10.0.0.20:5000;
        }
    }

Bilinmesi gerekenler:

- `proxy_pass` veya `return` gibi bir işleyicisi olmayan `location`, dosyaları
  `root` dizininden sunar (varsayılan olarak NGINX prefix'i altındaki `html/`).
- `proxy_pass` URL'sinin bir yol ile bitip bitmemesi, backend'in hangi yolu
  alacağını değiştirir. Yol yoksa istek yolu olduğu gibi iletilir.
- NGINX varsayılan olarak istemcinin `Host` başlığını ve adresini backend'e
  iletmez; backend'e ne gideceğini `proxy_set_header` belirler.

Kaynak: [proxy_pass](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass).

---

## Lab ortamı

Bu egzersiz NGINX'i Docker'da değil, doğrudan makinende çalıştırır:

- `backend_app.py` küçük bir HTTP backend'dir. Check onu **127.0.0.1:8080**
  üzerinde başlatır; senin çalıştırman gerekmez.
- `nginx.conf` senin dosyan. `events`/`http` yapısı, `logs/` altındaki loglar ve
  **8081** portunu dinleyen bir `server` hazır; `location /` boş.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) backend'i yalnızca
backend'in bildiği rastgele bir token ile başlatır, bu dizini prefix alarak
`nginx -t` çalıştırır, NGINX'i başlatır ve `http://127.0.0.1:8081/` ister. Cevap
HTTP 200 olmalı **ve** backend'in token'ını içermeli; yani NGINX'in kendi ürettiği
bir cevap geçmez. 8080 ve 8081 portları boş olmalı; değilse check bu portları
tutan süreci söyler.

---

## Görev

`nginx.conf` dosyasını, `location /` her isteği `127.0.0.1:8080` üzerindeki
backend'e iletecek şekilde düzenle. `listen 8081;` satırını ve dosyanın geri
kalanını olduğu gibi bırak.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- `http://127.0.0.1:8081/` HTTP 200 ile cevap vermeli.
- Cevap gövdesi backend'den gelmeli (backend'in token'ını içermeli).

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `got 404: location / has no handler`: location hâlâ boş; NGINX backend'e
  sormak yerine dosya arıyor.
- `got 502: NGINX tried to proxy, but nothing answered`: istek yanlış adrese ya da
  porta gitti. Backend hangi portu kullanıyor?
- `NGINX rejected the configuration: invalid URL prefix`: `proxy_pass` şema ile
  başlayan bir URL ister.
- `The response via NGINX does not come from the backend`: NGINX cevabı kendisi
  vermiş, örneğin bir `return` ile. Onu proxy işlemiyle değiştir.

---

## Pekiştirme soruları

- `location /app/` içinde, `proxy_pass` adresten sonra yol içermiyorsa `/app/x`
  isteği için backend hangi yolu alır; `/` ile bitiyorsa hangisini?
- Backend varsayılan olarak hangi `Host` başlığını alır ve istemcinin gönderdiği
  `Host`'u nasıl iletirsin?
- Backend yalnızca NGINX'ten gelen bağlantıları görür. İstemcinin IP adresini yine
  de nasıl öğrenebilir?
- Backend bağlantıyı reddettiğinde istemci ne alır ve NGINX'in bağlanmak ve cevabı
  okumak için ne kadar bekleyeceğini hangi direktifler sınırlar?
