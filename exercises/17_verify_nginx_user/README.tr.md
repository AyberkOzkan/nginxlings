# 17 – Verify NGINX User & Processes

## Amaç

Bu egzersizde NGINX'in süreç modelini öğrenecek, worker process'lerin hangi
hesapla çalışacağını ayarlayacak ve sonucu yapılandırmaya güvenmek yerine süreç
listesinden doğrulayacaksın.

---

## Arka plan – master ve worker'lar

- NGINX bir **master process** ve bir ya da daha fazla **worker process**
  başlatır. Master yapılandırmayı okur, dinleyen soketleri ve log dosyalarını açar,
  worker'ları başlatır, yeniden yükler ve durdurur. İstemci bağlantılarını
  worker'lar işler.
- Master, NGINX'i başlatan hesapla kalır (bu container'da root). **main**
  context'teki `user` direktifi, worker'ların geçeceği hesabı belirler; böylece
  güvenilmeyen trafiği işleyen süreçlerin yetkisi daha az olur. Biçimi
  `user <hesap> [<grup>];` şeklindedir; grup verilmezse hesapla aynı adlı grup
  kullanılır.
- Yapılandırmada `user` direktifi yoksa NGINX derlendiği hesabı kullanır;
  `nginx -V` bunu `configure arguments:` arasında gösterir. Açıkça yazmak,
  yapılandırmanın gerçekte olanı söylemesini sağlar.
- `user` yalnızca master root olarak çalıştığında işe yarar. Aksi hâlde NGINX,
  direktifin "makes sense only if the master process runs with super-user
  privileges" olduğunu söyleyip onu yok sayar.

Genel bir örnek (Debian'ın kendi `nginx.conf`'u farklı bir hesap kullanır):

    user www-data;
    worker_processes auto;

Kaynak: [user](https://nginx.org/en/docs/ngx_core_module.html#user),
[Controlling NGINX](https://nginx.org/en/docs/control.html).

---

## Lab ortamı

- `Dockerfile`, `nginx:1.28.3-alpine` üzerine kurulur ve container içinde
  `ps aux` çalışsın diye `procps` ekler. Image'da zaten bir `nginx` hesabı var.
- `nginx.conf` içinde 8080 portunda `process-user-ok` cevabı veren bir `server` ve
  en üstte, main context'in olduğu yerde bir `TODO` var.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i build eder,
`nginx -t` çalıştırır, container'ı başlatır (host portu 18170), içinde `nginx -T`
ile etkin yapılandırmayı okur, ardından `ps aux` çıktısını okur ve `/` ister.

---

## Görev

`nginx.conf` içinde worker process'lerin `nginx` hesabıyla (istersen `nginx`
grubuyla) çalışacağını açıkça belirt. Direktifi ait olduğu context'e koy.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- Etkin yapılandırmada `nginx` için etkin bir `user` direktifi olmalı.
- `ps aux`, sahibi `root` olan `nginx: master process`'i ve sahibi `nginx` olan en
  az bir `nginx: worker process`'i göstermeli.
- `/`, `process-user-ok` ile cevap vermeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `No active user directive for the nginx account yet`: direktif eksik, hâlâ yorum
  satırı ya da başka bir hesabı gösteriyor.
- `NGINX rejected the configuration: "user" directive is not allowed here`:
  direktif `events {}`, `http {}` ya da `server {}` içine konmuş. Geriye hangi
  context kalıyor?
- `getpwnam("...") failed`: bu hesap image'da yok. Yazımı kontrol et; image'ın
  hesabı `nginx`.
- `Could not find an nginx worker process running as the nginx user`: check'in
  yazdırdığı `ps aux` çıktısına bak. Worker'ların sahibi hangi hesap?

---

## Pekiştirme soruları

- `user` direktifini kaldırsaydın bu image'da worker'lar hangi hesapla çalışırdı
  ve bunun sebebini nerede görürsün?
- Master process neden root olarak kalır da worker'lar kalmaz?
- 8080 portundaki dinleyen soketi hangi süreç açar ve root olarak çalışmayan bir
  worker bu soketten nasıl bağlantı kabul edebilir?
- `ps aux`'ta kaç worker process görüyorsun ve bu sayıyı hangi direktif belirler?
