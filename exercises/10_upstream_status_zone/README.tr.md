# 10 – Upstream State in Shared Memory (zone up_mem)

## Amaç

Bu egzersizde üyeleri çalışma sırasında adlarıyla bulunan Docker container'ları
olan bir HTTP upstream kuracak, durumunu bir shared memory zone'da tutacak, her
isteği hangi backend'in cevapladığını gösterecek ve bir backend durduğunda NGINX'in
diğerine geçişini izleyeceksin.

Pratik edeceklerin:

- upstream `zone` direktifi,
- `resolve` ve Docker'ın DNS sunucusuyla çalışma sırasında ad çözümleme,
- `$upstream_addr` değerini bir response header'da göstermek,
- reddedilen bir bağlantının NGINX'e sıradaki sunucuyu nasıl denettirdiği.

---

## Arka plan – ortak durum, çalışma sırasında DNS ve failover

Zone olmadan her worker process bir upstream'in durumunun kendi kopyasını tutar:
round-robin sırası ve hata sayaçları. `upstream` bloğundaki `zone <ad> <boyut>;`,
grubun yapılandırmasını ve durumunu shared memory'ye taşır; böylece bütün worker'lar
aynı durumu görür.

Bir upstream'deki adlar normalde bir kez, yapılandırma yüklenirken çözülür.
`server` satırındaki `resolve` parametresi, `http` context'indeki `resolver`'ı
kullanarak adı NGINX çalışırken yeniden çözer. Bulunan adresler zone'da saklanır; bu
yüzden `resolve` yalnızca zone'u olan bir upstream'de çalışır. Açık kaynak NGINX
`resolve`'u 1.27.3'ten beri destekler; bu image 1.28.3 kullanır.

Genel bir örnek (bu lab'dakinden farklı ad, adres ve boyutlar):

    http {
        resolver 10.0.0.2 valid=30s;

        upstream api_pool {
            zone api_state 128k;
            server api-a.internal:9000 resolve;
            server api-b.internal:9000 resolve;
        }

        server {
            listen 80;

            location /api/ {
                proxy_pass http://api_pool;
            }
        }
    }

Bilinmesi gerekenler:

- Kullanıcı tanımlı bir Docker ağındaki container içinde Docker'ın gömülü DNS
  sunucusu `127.0.0.11` adresinde cevap verir. NGINX container'ı içindeki
  `127.0.0.1` NGINX'in kendisidir, backend'lerden biri asla değildir.
- `$upstream_addr`, cevabı veren sunucunun adresini tutar. Yeniden denemeden sonra
  denenen bütün adresleri virgülle ayırarak listeler.
- `add_header` bir response header ekler. `always` parametresi olmadan yalnızca
  başarılı ve yönlendirme cevaplarına eklenir (200, 201, 204, 206, 301, 302, 303,
  304, 307, 308).
- Bir bağlantı reddedildiğinde varsayılan `proxy_next_upstream error timeout;`
  isteği gruptaki sıradaki sunucuya verir.
- Açık kaynak NGINX upstream durumunu bir API ya da status sayfasıyla raporlamaz
  (bu bir NGINX Plus özelliğidir); bu yüzden bu lab seçilen backend'i bir response
  header ile görünür kılar.

Kaynak: [ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html)
(`zone`, `server ... resolve`), [resolver](https://nginx.org/en/docs/http/ngx_http_core_module.html#resolver),
[add_header](https://nginx.org/en/docs/http/ngx_http_headers_module.html#add_header).

---

## Lab ortamı

Check, tek bir Docker ağında üç container başlatır:

- **ex10_backend1:8081** ve **ex10_backend2:8082**: `backend=8081` ve `backend=8082`
  cevabı veren iki küçük BusyBox HTTP sunucusu.
- `Dockerfile`'ın `nginx:1.28.3-alpine` üzerine senin `nginx.conf`'unla kurduğu,
  `http://127.0.0.1:19100/` adresine açılan NGINX container'ı.

`nginx.conf` senin dosyan. Docker DNS için `resolver 127.0.0.11` satırı `http`
içinde hazır; `upstream backend` bloğunda bir yer tutucu sunucu var; `location /`
hâlâ yer tutucu metin döndürüyor.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i
(`nginxlings/10_upstream_status_zone:check`) build eder ve `nginx -t`'yi **backend
container'ları henüz yokken** çalıştırır. Ardından her şeyi başlatır, 10 istek
gönderip `X-Upstream` başlığını okur, `ex10_backend2` içindeki HTTP sunucusunu
durdurur (container ayakta kalır, bu yüzden 8082'ye bağlantılar reddedilir) ve 12
istek daha gönderir.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `upstream backend` içine **up_mem** adında, **64k** boyutunda bir shared memory
   zone ekle.
2. Yer tutucu sunucuyu, ikisi de çalışma sırasında çözülen `ex10_backend1:8081` ve
   `ex10_backend2:8082` ile değiştir.
3. `location /`, metin döndürmek yerine istekleri `backend` upstream'ine proxy etsin.
4. Değeri `$upstream_addr` olan bir **X-Upstream** response header'ı ekle.
5. `resolver` satırını koru.

---

## Başarı kriterleri

- Image build olmalı ve `nginx -t` yapılandırmayı kabul etmeli.
- Aktif yapılandırmada 64k'lık `up_mem` zone'u ve `$upstream_addr` değerli
  `X-Upstream` başlığı bulunmalı.
- Durdurmadan önce 10 isteğin `X-Upstream` değerlerinde hem 8081 hem 8082
  görünmeli.
- Durdurmadan sonra 12 isteğin tamamı 8081 üzerinden `200` dönmeli.
  `<addr>:8082, <addr>:8081` gibi bir yeniden deneme zinciri sorun değildir.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `No shared memory zone named up_mem with a size of 64k`: başlangıç durumu bu. Ad
  ve boyut birebir aynı olmalı.
- `resolving names at run time requires upstream "backend" ... to be in shared
  memory`: `resolve` var ama zone yok.
- `host not found in upstream "ex10_backend1:8081"`: ad, yapılandırma yüklenirken
  çözülmüş ve backend container'ları o anda henüz yok. Hangi `server` parametresi
  çözümlemeyi erteler?
- `host not found in upstream "ex10_backend1"`: `proxy_pass`, upstream grubu yerine
  doğrudan bir container'ı gösteriyor.
- `no resolver defined to resolve names at run time`: `resolver` satırı silinmiş.
- `No active X-Upstream response header`: başlık yok, yorum satırında ya da başka
  bir değere ayarlanmış.
- `Before stopping, expected both upstream ports in X-Upstream. Got 8081=0,
  8082=0`: cevaplarda upstream adresi yok. `location /` proxy etmek yerine hâlâ metin
  mi döndürüyor?
- `After stopping backend2, 1 of 12 requests did not succeed through 8081`:
  durdurmadan sonraki ilk istek yeniden denenmek yerine `502` almış. Sıradaki
  sunucuyu deneme kapatılmış mı?

---

## Pekiştirme soruları

- 8082'ye ilk bağlantı reddedildikten sonra gelen istekler doğrudan 8081'e gider.
  Buna hangi `server` parametreleri karar verir ve ne kadar süreyle?
- Check HTTP sunucusunu durdurur ama backend container'ını ayakta tutar. Container
  silinseydi istemci ne fark ederdi?
- `add_header`'da `always` olmasaydı, `502` cevabında `X-Upstream` başlığı ne
  gösterirdi?
- Bir backend container'ı yeniden oluşturuluyor ve yeni bir IP adresi alıyor.
  `resolve` ile trafiğe ne olur, `resolve` olmasaydı ne olurdu?
