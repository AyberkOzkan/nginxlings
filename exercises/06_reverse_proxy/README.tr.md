# 06 – HTTP Ters Vekil Sunucu Temelleri

## Amaç

Bu egzersizde NGINX'i küçük bir arka uç sunucunun (backend) önüne **HTTP ters vekil
sunucu** (reverse proxy) olarak koyacaksın: istemciler yalnızca NGINX ile konuşur,
yanıtı arka uç sunucudan NGINX getirir.

---

## Arka plan – ters vekil sunucu ne yapar

Ters vekil sunucu, istemcinin bağlantısını kabul eder, arka uç sunucuya kendi
bağlantısını açar, isteği iletir ve arka uç sunucunun yanıtını geri döndürür.
İstemci arka ucun adresini hiç görmez.

NGINX'te bu iş bir `location` içindeki `proxy_pass`'e düşer. `proxy_pass`, şemalı
(`http://` veya `https://`) bir URL ve bir adres alır. Genel bir örnek (bu
laboratuvardakinden farklı ad ve adresler):

    server {
        listen 80;

        location /shop/ {
            proxy_pass http://10.0.0.20:5000;
        }
    }

Bilinmesi gerekenler:

- `proxy_pass` veya `return` gibi bir işleyicisi olmayan `location`, dosyaları
  `root` dizininden sunar (varsayılan olarak NGINX önekinin (prefix) altındaki
  `html/`).
- `proxy_pass` URL'sinin bir yol ile bitip bitmemesi, arka ucun hangi yolu
  alacağını değiştirir. Yol yoksa istek yolu olduğu gibi iletilir.
- NGINX varsayılan olarak istemcinin `Host` başlığını (header) ve adresini arka uca
  iletmez; arka uca ne gideceğini `proxy_set_header` belirler.

Kaynak: [proxy_pass](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass).

---

## Laboratuvar ortamı

Bu egzersiz NGINX'i Docker'da değil, doğrudan makinende çalıştırır:

- `backend_app.py` küçük bir HTTP arka uç sunucusudur. Kontrol onu
  **127.0.0.1:8080** üzerinde başlatır; senin çalıştırman gerekmez.
- `nginx.conf` senin dosyan. `events`/`http` yapısı, `logs/` altındaki günlükler
  (log) ve **8081** portunu dinleyen bir `server` hazır; `location /` boş.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) arka uç sunucuyu yalnızca
onun bildiği rastgele bir belirteçle (token) başlatır, bu dizini önek alarak
`nginx -t` çalıştırır, NGINX'i başlatır ve `http://127.0.0.1:8081/` ister. Yanıt
HTTP 200 olmalı **ve** arka ucun belirtecini içermeli; yani NGINX'in kendi ürettiği
bir yanıt geçmez. 8080 ve 8081 portları boş olmalı; değilse kontrol bu portları
tutan süreci söyler.

---

## Görev

`nginx.conf` dosyasını, `location /` her isteği `127.0.0.1:8080` üzerindeki
arka uç sunucuya iletecek şekilde düzenle. `listen 8081;` satırını ve dosyanın geri
kalanını olduğu gibi bırak.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- `http://127.0.0.1:8081/` HTTP 200 ile yanıt vermeli.
- Yanıt gövdesi arka uçtan gelmeli (arka ucun belirtecini içermeli).

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `got 404: location / has no handler`: `location` hâlâ boş; NGINX arka uca
  sormak yerine dosya arıyor.
- `got 502: NGINX tried to proxy, but nothing answered`: istek yanlış adrese ya da
  porta gitti. Arka uç hangi portu kullanıyor?
- `NGINX rejected the configuration: invalid URL prefix`: `proxy_pass` şema ile
  başlayan bir URL ister.
- `The response via NGINX does not come from the backend`: NGINX yanıtı kendisi
  vermiş, örneğin bir `return` ile. Onu iletme işlemiyle değiştir.

---

## Pekiştirme soruları

- `location /app/` içinde, `proxy_pass` adresten sonra yol içermiyorsa `/app/x`
  isteği için arka uç hangi yolu alır; `/` ile bitiyorsa hangisini?
- Arka uç varsayılan olarak hangi `Host` başlığını alır ve istemcinin gönderdiği
  `Host`'u nasıl iletirsin?
- Arka uç yalnızca NGINX'ten gelen bağlantıları görür. İstemcinin IP adresini yine
  de nasıl öğrenebilir?
- Arka uç bağlantıyı reddettiğinde istemci ne alır ve NGINX'in bağlanmak ve yanıtı
  okumak için ne kadar bekleyeceğini hangi direktifler sınırlar?
