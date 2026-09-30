# 12 – Logs & Variables

## Amaç

Bu egzersizde kendi access log formatını tanımlayacak, onu NGINX değişkenleriyle
dolduracak ve log'u container'ın standart çıktısına göndereceksin.

Pratik edeceklerin:

- `log_format` ve ait olduğu context,
- adı verilmiş bir formatla `access_log`,
- sık kullanılan istek değişkenleri ve değerlerinin nereden geldiği.

---

## Arka plan – log formatları ve değişkenler

`log_format <ad> '<metin>';`, `http` context'inde adlandırılmış bir format tanımlar.
Metin düz yazıyı değişkenlerle karıştırır; her istek, her değişkenin yerine değerinin
yazıldığı bir satır üretir. `access_log <path> <format>;` log'u o formatla açar.

Genel bir örnek (bu lab'dakinden farklı ad, değişken ve dosya):

    log_format timing '$remote_addr [$time_local] "$request" rt=$request_time';

    server {
        access_log /var/log/nginx/timing.log timing;
    }

Bilinmesi gerekenler:

- Bir format, onu kullanan `access_log`'dan önce tanımlanmalıdır; ad, yapılandırma
  okunurken aranır.
- Format adı verilmeyen `access_log` önceden tanımlı `combined` formatını kullanır.
  Hiç `access_log` olmasa bile NGINX, derlendiği path'e `combined` formatında log
  yazar.
- İç context'teki `access_log off;` orada log'u kapatır.
- Göreli bir log path'i, egzersiz 01'deki `root` gibi NGINX prefix'ine göre çözülür.
- Bazı değişkenler: `$remote_addr` bağlantının istemci adresi, `$status` cevabın
  durum kodu, `$request` isteğin tam satırıdır (method, URI, protokol).
  `$http_<ad>` herhangi bir istek başlığını okur: ad küçük harfle, tireler alt
  çizgiyle yazılır.
- Container içinde `/dev/stdout`'a yazılan log'lar `docker logs` içinde görünür.

Kaynak: [ngx_http_log_module](https://nginx.org/en/docs/http/ngx_http_log_module.html),
[variables](https://nginx.org/en/docs/varindex.html).

---

## Lab ortamı

Her şey bu dizinden build edilen tek bir Docker container içinde çalışır:

- `Dockerfile`, image'i `nginx:1.28.3-alpine` üzerine senin `nginx.conf`'unla kurar.
- `nginx.conf` senin dosyan. Server'ı 8080 portunda `/` için `200` döner; TODO
  yorumları format ve access log için gerekenleri listeler.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i
(`nginxlings/12_logs_and_variables:check`) build eder, aktif yapılandırmayı
`nginx -T` ile okur, container'ın 8080 portunu `http://127.0.0.1:18120/` adresine
açar, `nginxlings-test-agent/1.0` User-Agent'ı ile beş istek gönderir ve bunları
`docker logs` içinde arar.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `http` içinde en az `$remote_addr`, `$status`, `$request` ve `$http_user_agent`
   içeren **main** adında bir log formatı tanımla.
2. Access log'u `main` formatında **/dev/stdout**'a yaz.
3. 8080 portunda `/` için `200` dönen server'ı koru.

---

## Başarı kriterleri

- `nginx -T`, dört değişkeni içeren aktif bir `log_format main` ve `main`'i kullanan
  aktif bir `access_log` göstermeli.
- Test isteklerinden sonra `docker logs`, test User-Agent'ını, `GET /` isteğini,
  `200` durum kodunu ve bir IP adresini içeren satırlar içermeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `No active 'log_format main ...;' found`: başlangıç durumu bu.
- `log_format main does not include $...`: formatta mesajda adı geçen değişken
  eksik.
- `No active access_log uses the main format yet`: format adı verilmeyen bir
  `access_log` `main` değil `combined` yazar.
- `unknown log format "main"`: ad bir yerde yanlış yazılmış, format onu kullanan
  `access_log`'dan sonra tanımlanmış ya da `http` içinde değil.
- `open() "/etc/nginx/logs/access.log" failed (2: No such file or directory)`: göreli
  bir path `/etc/nginx` prefix'ine göre çözülmüş.
- `No access log line contains the test User-Agent`: log `/dev/stdout`'a mı
  yazılıyor? Server içinde log kapatılmış mı?

---

## Pekiştirme soruları

- Bu image'da `access_log /var/log/nginx/access.log main;` de check'ten geçer. Bu
  satırlar neden `docker logs`'a ulaşır?
- `index.html` üzerinden cevaplanan `GET /?a=1` isteği için `$request`,
  `$request_uri` ve `$uri` ne içerir?
- Bir load balancer'ın arkasında `$remote_addr` ne gösterir ve NGINX'in bunun yerine
  asıl istemci adresini log'lamasını hangi modül sağlar?
