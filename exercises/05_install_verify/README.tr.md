# 05 – Resmi Depodan Kurulum ve Doğrulama

## Amaç

Bu egzersizde NGINX'i dağıtımın paket deposundan bir konteyner imajına (container
image) kuracak, sonra kurulumu yeni bir sunucuda yapacağın gibi doğrulayacaksın:
hangi sürüm, nasıl derlenmiş, hangi paketten gelmiş ve HTTP sunuyor mu.

---

## Arka plan – paketler ve imajlar

- Debian'ın paket aracı `apt-get`'tir. Yeni bir `debian:*-slim` imajı **paket
  dizini olmadan** gelir; dizin indirilmeden hiçbir paket bulunamaz.
- `docker build` sırasında hiçbir soruya yanıt verilemez. `apt-get`'in kendi onay
  sorusuna otomatik "evet" diyen bir seçeneği vardır;
  `DEBIAN_FRONTEND=noninteractive` ortam değişkeni de debconf'a yapılandırma
  diyaloğu açmamasını söyler.
- Her `RUN` talimatı bir imaj katmanı olur ve Docker katmanları önbelleğe (cache)
  alır. Dizini yenilemek ile paket kurmak ayrı `RUN` talimatlarında olursa, sonraki
  bir oluşturma işlemi (build) eski önbellekteki dizini yeni bir kurulum satırıyla
  kullanabilir. Bu yüzden ikisi genellikle tek bir `RUN` içinde `&&` ile zincirlenir.
- `/var/lib/apt/lists/` altına indirilen dizin çalışma zamanında gerekmez. Onu
  **aynı `RUN` içinde** silmek katmanı küçük tutar; sonraki bir `RUN` içinde
  silmek işe yaramaz, çünkü önceki katman onu hâlâ içerir.
- Doğrulama komutları:
  - `nginx -v` sürümü yazar.
  - `nginx -V` ayrıca derleyiciyi ve NGINX'in hangi `configure arguments:` ile
    derlendiğini yazar; buradan hangi modüllerin olduğunu ve statik mi dinamik
    mi derlendiklerini görürsün.
  - `dpkg -s nginx` paketin durumunu, sürümünü ve bağımlılıklarını gösterir.

Buradaki "resmi depo", taban imajın zaten kullandığı Debian paket arşividir.
nginx.org da daha yeni NGINX sürümleri içeren kendi paket deposunu yayınlar; onu
eklemek için ek bir apt kaynağı ve imza anahtarı gerekir ve bu egzersiz onu
kullanmaz. Kontrol, `dpkg`'nin kurulu olarak gösterdiği her `nginx` paketini kabul
eder.

Kaynak: [Installing NGINX Open Source](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-open-source/).

---

## Laboratuvar ortamı

- `Dockerfile`, `debian:12.15-slim` ile başlar. Kurulumun yeri bir `TODO` yorumuyla
  işaretli; `EXPOSE 80` ve NGINX'i ön planda çalıştıran `CMD` hazır.
- Debian paketi 80 portunda yanıt veren varsayılan bir site ile gelir; bu yüzden
  bu egzersizde kendi NGINX yapılandırmanı yazman gerekmez.

Kontrol (Nginxlings'te `c` veya Enter ya da `./check.sh`) imajı
(`nginxlings/05_install_verify:check`) oluşturur, `nginx`'in `PATH` üzerinde
olduğunu doğrular, `nginx -v` ve `nginx -V` çalıştırır (`configure arguments:`
satırını arar), `dpkg`'nin `nginx` paketini kurulu gösterdiğini doğrular, ardından
80 portunu boş bir yerel porta açarak bir konteyner başlatır ve `/` ister.

---

## Görev

`Dockerfile` içindeki `TODO`'nun altına, şunları yapan tek bir `RUN` talimatı
ekle:

1. paket dizinini yenilesin,
2. `nginx` paketini hiçbir soru sormadan kursun,
3. indirilen paket dizinini silsin.

`FROM`, `EXPOSE` ve `CMD` satırlarını olduğu gibi bırak.

---

## Başarı kriterleri

- İmaj oluşmalı.
- `nginx` `PATH` üzerinde olmalı; `nginx -v` ve `nginx -V` çalışmalı, `nginx -V`
  `configure arguments:` satırını göstermeli.
- `dpkg -s nginx`, `Status: install ok installed` göstermeli.
- İmajdan başlatılan bir konteyner, 80 portunda `/` isteğine başarılı bir HTTP
  yanıtı vermeli.

---

## Takıldıysan

Bunları sırayla dene, aralarda kontrolü yeniden çalıştır:

- `E: Unable to locate package nginx`: paket dizini boş. Kurulumdan önce ne
  çalışmalı?
- `Do you want to continue? [Y/n] Abort.`: kurulum, imaj oluşturulurken hiç
  gelmeyecek bir yanıtı bekliyor.
- Oluşturma çıktısında `debconf: unable to initialize frontend: Dialog`: yalnızca
  bir uyarı, kurulum yine çalışır. Arka plan bölümündeki ortam değişkeni bu uyarıyı
  susturur.
- `The image has no nginx yet.`: imaj oluştu ama NGINX'i hiçbir şey kurmamış.
  `RUN` talimatın `TODO`'nun altında mı ve yorum satırı olarak kalmadı mı?

---

## Pekiştirme soruları

- Debian paketinde `stream` modülünün olup olmadığını `nginx -V` çıktısının neresinde
  görürsün; modül statik mi derlenmiş, dinamik mi? Bir `stream {}` bloğunun
  çalışması için başka neye ihtiyacın olur?
- Dizini yenilemek ile paket kurmak neden genellikle tek bir `RUN` talimatında
  zincirlenir?
- Debian 12 hangi NGINX sürümünü sunar ve güncel sürümü nginx.org'dan kurmak için
  neyi değiştirirdin?
- Debian paketi varsayılan siteyi nerede tutar ve bunu `nginx.conf` içindeki hangi
  satır yükler? Diğer egzersizlerde kullanılan resmi `nginx` Docker imajından
  farkı ne?
