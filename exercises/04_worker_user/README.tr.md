# 04 – İşçi Süreç Kullanıcısı ve İzinler

## Amaç

Bu egzersizde NGINX'e, dosyalarını yalnızca tek bir hesabın okuyabildiği bir siteyi
sunduracaksın. İşçi süreçlerin (worker process) hangi hesapla çalıştığını, `200` ile
`403` arasındaki farkı bunun neden belirlediğini ve `user` direktifinin bunu nasıl
değiştirdiğini göreceksin.

Pratik edeceklerin:

- `user` direktifi ve ait olduğu bağlam (context),
- bir dosyayı kimin açabileceğini sahip, grup ve izin kipi (mode) bitlerinden
  okumak,
- bir `403 Forbidden`'ın sebebini hata günlüğünde (error log) bulmak.

---

## Arka plan – dosyaları hangi hesap açar

NGINX bir **ana süreç** (master process) ve birkaç **işçi süreç** başlatır. Ana süreç
yapılandırmayı okur ve dinleyen soketleri (socket) açar; istekleri işçi süreçler
işler ve sundukları dosyaları da onlar açar. Bu yüzden dosya erişimi ana sürecin değil,
**işçi sürecin** hesabına göre denetlenir. (Egzersiz 17 süreç modelini daha ayrıntılı
ele alır.)

**main** bağlamındaki `user` direktifi bu hesabı ve istenirse grubunu belirler:

    user www-data;
    user app appgroup;

Grup verilmezse NGINX hesapla aynı adlı grubu kullanır. Hesap sistemde (`/etc/passwd`
içinde) bulunmalıdır, yoksa `nginx -t` hata verir. `user` direktifi yoksa NGINX
derlendiği hesabı kullanır.

`/var/www/blog/index.html` dosyasını sunabilmek için işçi sürecin yol üzerindeki her
dizinde **çalıştırma** (`x`), dosyada ise **okuma** (`r`) iznine ihtiyacı vardır.
Unix, süreç dosyanın sahibiyse sahip bitlerine, değilse ve süreç dosyanın grubundaysa
grup bitlerine, ikisi de değilse "diğerleri" bitlerine bakar. Genel bir örnek:

    $ ls -ld /var/www/blog /var/www/blog/index.html
    drwxr-x---  2 blog blog 4096 Jan  1 10:00 /var/www/blog
    -rw-r-----  1 blog blog   12 Jan  1 10:00 /var/www/blog/index.html

Burada sahip olan `blog` dizine girip dosyayı okuyabilir, `blog` grubunun üyeleri de
aynısını yapabilir, diğer hesaplar ise ikisini de yapamaz.

Bilinmesi gerekenler:

- Bir işçi süreç dosyayı açamadığında NGINX `403 Forbidden` döner ve sebebini
  `(13: Permission denied)` ile birlikte hata günlüğüne yazar.
- İşçi süreçleri `root` ile çalıştırmak bütün izin sorunlarını ortadan kaldırır; tam da
  bu yüzden bir çözüm değildir: işçi süreçteki herhangi bir hata tam yetkiyle çalışır.
- `user` yalnızca ana süreç root olarak çalıştığında etkilidir. Aksi hâlde NGINX onu
  bir uyarıyla yok sayar. Bu laboratuvarın, ana sürecin root olduğu bir konteyner
  (container) içinde çalışmasının sebebi budur.

Kaynak: [user](https://nginx.org/en/docs/ngx_core_module.html#user),
[Beginner's guide](https://nginx.org/en/docs/beginners_guide.html).

---

## Laboratuvar ortamı

Her şey bu dizinden oluşturulan (build) tek bir Docker konteyneri içinde çalışır:

- `Dockerfile`, imajı (image) `nginx:1.28.3-alpine` üzerine kurar. Ayrı bir hesap ve
  bu hesaba ait, diğerlerine kapalı, tek bir `index.html` içeren **/srv/site** sitesini
  oluşturur. **Dockerfile'ı olduğu gibi bırak**; kontrol iki yolun sahibini ve
  iznini doğrular.
- `nginx.conf` senin dosyan. `/srv/site`'ı 8080 portunda sunar ve bir Debian
  sunucusundan kopyalanmış bir `user` satırıyla başlar.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) imajı
(`nginxlings/04_worker_user:check`) oluşturur, `nginx -t` çalıştırır,
`/srv/site` ve `/srv/site/index.html`'in sahibini ve iznini Dockerfile'dakiyle
karşılaştırır, konteynerin 8080 portunu `http://127.0.0.1:18040/` adresine açar,
işçi süreçleri `ps -o user,args` ile listeler ve `/` ister.

---

## Görev

Yalnızca `nginx.conf` dosyasını düzenle:

1. İşçi süreçleri, bu imajda var olan ve `/srv/site/index.html`'i okuyabilen bir
   hesapla çalıştır.
2. İşçi süreçler `root` ile çalışmamalı.
3. `server` bloğunu koru: 8080 portu, `root /srv/site`.

---

## Başarı kriterleri

- İmaj oluşmalı ve `nginx -t` yapılandırmayı kabul etmeli.
- `/srv/site` ve `/srv/site/index.html`, Dockerfile'ın verdiği sahip, grup ve izni
  korumalı.
- En az bir işçi süreç çalışmalı ve hiçbiri `root` ile çalışmamalı.
- `/`, `/srv/site/index.html` sayfasıyla `200` dönmeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `getpwnam("www-data") failed`: başlangıç durumu bu. `www-data`, Debian ve
  Ubuntu'daki web sunucusu hesabıdır. Bu Alpine imajında hangi hesaplar var?
- `"user" directive is not allowed here`: direktif bir bloğun içinde. Geriye hangi
  bağlam kalıyor?
- `403 Forbidden: the workers cannot read /srv/site` ve hata günlüğünde
  `is forbidden (13: Permission denied)`: kontrolün yazdırdığı işçi süreç hesabını
  dosyaların sahibi ve grubuyla karşılaştır. Dockerfile'daki `chown` ve `chmod`
  satırları bunları gösterir. Kendin bakmak için bir imaj oluştur
  (`docker build -t ex04-debug .`) ve
  `docker run --rm ex04-debug ls -ld /srv/site /srv/site/index.html` çalıştır.
- `The workers run as root`: root her şeyi okuyabilir, bu yüzden izinler hakkında
  hiçbir şey kanıtlamaz. Yetkisiz bir hesap seç.
- `The owner or mode of /srv/site changed`: Dockerfile değiştirilmiş. Eski hâline
  getir ve görevi `nginx.conf` içinde çöz.

---

## Pekiştirme soruları

- Bu kontrolden birden fazla `user` satırı geçer. Sahip bitlerine dayanan birini ve
  grup bitlerine dayanan birini bul. Hangisini seçerdin, neden?
- Bir dizin neden yalnızca `r` değil `x` de ister? `/srv/site`'ın izni `0740`
  olsaydı, dosyanın grubundaki bir işçi süreç ne alırdı?
- Konteyner `docker run --user 101 ...` ile başlatılırsa `user` direktifine ne olur?
  Bu durumda başka ne başarısız olur?
- `nginx.conf`'a ve dosya izinlerine dokunmadan, bir kurulum varsayılan `nginx`
  hesabına `/srv/site`'ı okuma yetkisini nasıl verebilir?
