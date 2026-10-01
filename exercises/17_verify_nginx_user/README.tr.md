# 17 – NGINX Kullanıcısını ve Süreçlerini Doğrulama

## Amaç

Bu egzersizde NGINX'in süreç modelini öğrenecek, işçi süreçlerin (worker process)
hangi hesapla çalışacağını ayarlayacak ve sonucu yapılandırmaya güvenmek yerine
süreç listesinden doğrulayacaksın.

---

## Arka plan – ana süreç ve işçi süreçler

- NGINX bir **ana süreç** (master process) ve bir ya da daha fazla **işçi süreç**
  başlatır. Ana süreç yapılandırmayı okur, dinleyen soketleri ve günlük (log)
  dosyalarını açar, işçi süreçleri başlatır, yeniden yükler ve durdurur. İstemci
  bağlantılarını işçi süreçler işler.
- Ana süreç, NGINX'i başlatan hesapla kalır (bu konteynerde (container) root).
  **main** bağlamındaki (context) `user` direktifi, işçi süreçlerin geçeceği hesabı
  belirler; böylece güvenilmeyen trafiği işleyen süreçlerin yetkisi daha az olur.
  Biçimi `user <hesap> [<grup>];` şeklindedir; grup verilmezse hesapla aynı adlı
  grup kullanılır.
- Yapılandırmada `user` direktifi yoksa NGINX derlendiği hesabı kullanır;
  `nginx -V` bunu `configure arguments:` arasında gösterir. Açıkça yazmak,
  yapılandırmanın gerçekte olanı söylemesini sağlar.
- `user` yalnızca ana süreç root olarak çalıştığında işe yarar. Aksi hâlde NGINX,
  direktifin "makes sense only if the master process runs with super-user
  privileges" olduğunu söyleyip onu yok sayar.

Genel bir örnek (Debian'ın kendi `nginx.conf`'u farklı bir hesap kullanır):

    user www-data;
    worker_processes auto;

Kaynak: [user](https://nginx.org/en/docs/ngx_core_module.html#user),
[Controlling NGINX](https://nginx.org/en/docs/control.html).

---

## Laboratuvar ortamı

- `Dockerfile`, `nginx:1.28.3-alpine` üzerine kurulur ve konteyner içinde
  `ps aux` çalışsın diye `procps` ekler. İmajda (image) zaten bir `nginx` hesabı var.
- `nginx.conf` içinde 8080 portunda `process-user-ok` yanıtı veren bir `server` ve
  en üstte, main bağlamının olduğu yerde bir `TODO` var.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) imajı oluşturur (build),
`nginx -t` çalıştırır, konteyneri başlatır (ana makine (host) portu 18170), içinde
`nginx -T` ile etkin yapılandırmayı okur, ardından `ps aux` çıktısını okur ve `/`
ister.

---

## Görev

`nginx.conf` içinde işçi süreçlerin `nginx` hesabıyla (istersen `nginx`
grubuyla) çalışacağını açıkça belirt. Direktifi ait olduğu bağlama koy.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- Etkin yapılandırmada `nginx` için etkin bir `user` direktifi olmalı.
- `ps aux`, sahibi `root` olan `nginx: master process`'i ve sahibi `nginx` olan en
  az bir `nginx: worker process`'i göstermeli.
- `/`, `process-user-ok` ile yanıt vermeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `No active user directive for the nginx account yet`: direktif eksik, hâlâ yorum
  satırı ya da başka bir hesabı gösteriyor.
- `NGINX rejected the configuration: "user" directive is not allowed here`:
  direktif `events {}`, `http {}` ya da `server {}` içine konmuş. Geriye hangi
  bağlam kalıyor?
- `getpwnam("...") failed`: bu hesap imajda yok. Yazımı kontrol et; imajın
  hesabı `nginx`.
- `Could not find an nginx worker process running as the nginx user`: kontrolün
  yazdırdığı `ps aux` çıktısına bak. İşçi süreçlerin sahibi hangi hesap?

---

## Pekiştirme soruları

- `user` direktifini kaldırsaydın bu imajda işçi süreçler hangi hesapla çalışırdı
  ve bunun sebebini nerede görürsün?
- Ana süreç neden root olarak kalır da işçi süreçler kalmaz?
- 8080 portundaki dinleyen soketi hangi süreç açar ve root olarak çalışmayan bir
  işçi süreç bu soketten nasıl bağlantı kabul edebilir?
- `ps aux`'ta kaç işçi süreç görüyorsun ve bu sayıyı hangi direktif belirler?
