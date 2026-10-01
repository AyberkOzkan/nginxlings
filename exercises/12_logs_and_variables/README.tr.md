# 12 – Günlükler ve Değişkenler

## Amaç

Bu egzersizde kendi erişim günlüğü biçimini (access log format) tanımlayacak, onu
NGINX değişkenleriyle dolduracak ve günlüğü konteynerin (container) standart çıktısına
göndereceksin.

Pratik edeceklerin:

- `log_format` ve ait olduğu bağlam (context),
- adı verilmiş bir biçimle `access_log`,
- sık kullanılan istek değişkenleri ve değerlerinin nereden geldiği.

---

## Arka plan – günlük biçimleri ve değişkenler

`log_format <ad> '<metin>';`, `http` bağlamında adlandırılmış bir biçim tanımlar.
Metin düz yazıyı değişkenlerle karıştırır; her istek, her değişkenin yerine değerinin
yazıldığı bir satır üretir. `access_log <yol> <biçim>;` günlüğü o biçimle açar.

Genel bir örnek (bu laboratuvardakinden farklı ad, değişken ve dosya):

    log_format timing '$remote_addr [$time_local] "$request" rt=$request_time';

    server {
        access_log /var/log/nginx/timing.log timing;
    }

Bilinmesi gerekenler:

- Bir biçim, onu kullanan `access_log`'dan önce tanımlanmalıdır; ad, yapılandırma
  okunurken aranır.
- Biçim adı verilmeyen `access_log` önceden tanımlı `combined` biçimini kullanır.
  Hiç `access_log` olmasa bile NGINX, derlendiği yola `combined` biçiminde günlük
  yazar.
- İç bağlamdaki `access_log off;` orada günlüğü kapatır.
- Göreli bir günlük yolu, egzersiz 01'deki `root` gibi NGINX önekine (prefix) göre
  çözülür.
- Bazı değişkenler: `$remote_addr` bağlantının istemci adresi, `$status` yanıtın
  durum kodu, `$request` isteğin tam satırıdır (yöntem, URI, protokol).
  `$http_<ad>` herhangi bir istek başlığını (header) okur: ad küçük harfle, tireler
  alt çizgiyle yazılır.
- Konteyner içinde `/dev/stdout`'a yazılan günlükler `docker logs` içinde görünür.

Kaynak: [ngx_http_log_module](https://nginx.org/en/docs/http/ngx_http_log_module.html),
[variables](https://nginx.org/en/docs/varindex.html).

---

## Laboratuvar ortamı

Her şey bu dizinden oluşturulan (build) tek bir Docker konteyneri içinde çalışır:

- `Dockerfile`, imajı (image) `nginx:1.28.3-alpine` üzerine senin `nginx.conf`'unla
  kurar.
- `nginx.conf` senin dosyan. `server`'ı 8080 portunda `/` için `200` döner; TODO
  yorumları biçim ve erişim günlüğü için gerekenleri listeler.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) imajı
(`nginxlings/12_logs_and_variables:check`) oluşturur, aktif yapılandırmayı
`nginx -T` ile okur, konteynerin 8080 portunu `http://127.0.0.1:18120/` adresine
açar, `nginxlings-test-agent/1.0` User-Agent'ı ile beş istek gönderir ve bunları
`docker logs` içinde arar.

---

## Görev

`nginx.conf` dosyasını düzenle:

1. `http` içinde en az `$remote_addr`, `$status`, `$request` ve `$http_user_agent`
   içeren **main** adında bir günlük biçimi tanımla.
2. Erişim günlüğünü `main` biçiminde **/dev/stdout**'a yaz.
3. 8080 portunda `/` için `200` dönen `server`'ı koru.

---

## Başarı kriterleri

- `nginx -T`, dört değişkeni içeren aktif bir `log_format main` ve `main`'i kullanan
  aktif bir `access_log` göstermeli.
- Test isteklerinden sonra `docker logs`, test User-Agent'ını, `GET /` isteğini,
  `200` durum kodunu ve bir IP adresini içeren satırlar içermeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `No active 'log_format main ...;' found`: başlangıç durumu bu.
- `log_format main does not include $...`: biçimde mesajda adı geçen değişken
  eksik.
- `No active access_log uses the main format yet`: biçim adı verilmeyen bir
  `access_log` `main` değil `combined` yazar.
- `unknown log format "main"`: ad bir yerde yanlış yazılmış, biçim onu kullanan
  `access_log`'dan sonra tanımlanmış ya da `http` içinde değil.
- `open() "/etc/nginx/logs/access.log" failed (2: No such file or directory)`: göreli
  bir yol `/etc/nginx` önekine göre çözülmüş.
- `No access log line contains the test User-Agent`: günlük `/dev/stdout`'a mı
  yazılıyor? `server` içinde günlük kapatılmış mı?

---

## Pekiştirme soruları

- Bu imajda `access_log /var/log/nginx/access.log main;` de kontrolden geçer. Bu
  satırlar neden `docker logs`'a ulaşır?
- `index.html` üzerinden yanıtlanan `GET /?a=1` isteği için `$request`,
  `$request_uri` ve `$uri` ne içerir?
- Bir yük dengeleyicinin (load balancer) arkasında `$remote_addr` ne gösterir ve
  NGINX'in bunun yerine asıl istemci adresini günlüğe yazmasını hangi modül sağlar?
