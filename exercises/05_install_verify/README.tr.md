# 05 – Resmi Depodan Kurulum ve Doğrulama

## Amaç

Bu egzersizde:

- NGINX’i bir container içinde **resmi işletim sistemi paket deposundan** (apt)
  kullanarak kuracaksın.
- Bu dizindeki `Dockerfile` ile bir imaj build edeceksin.
- İmaj içinde NGINX’in doğru kurulduğunu aşağıdaki komutlarla doğrulayacaksın:
  - `nginx -v`
  - `nginx -V`
- Container içinde NGINX’in çalışıp 80 numaralı porttan HTTP cevap verebildiğinden
  emin olacaksın.

---

## Arkaplan – resmi paket deposu

Debian/Ubuntu tabanlı sistemlerde NGINX genellikle resmi dağıtım depolarından,
örneğin:

    apt-get update
    apt-get install -y nginx

komutları ile kurulur.

Bu sayede:

- Paketlenmiş bir NGINX binary’si (`/usr/sbin/nginx` gibi),
- `/etc/nginx/` altında varsayılan konfigürasyonlar,
- Log ve diğer entegrasyonlar,

otomatik olarak sağlanır.

Bu egzersizde aynı kurulumu, bu dizindeki `Dockerfile` ile tanımlanan bir
container imajının içinde gerçekleştireceksin.

---

## Bu egzersizde ne yapıyorsun?

Çalışma dizinin:

    05_install_verify/

Buradaki temel dosya:

    Dockerfile

Yapman gerekenler:

1. Bu dizindeki `Dockerfile` dosyasını aç.

2. İçindeki `TODO` yorumunun altına, aşağıdaki adımları yapan bir veya birden
   fazla `RUN` komutu ekle:

    - Paket indeksini güncelle (`apt-get update`),
    - Resmi repodan `apt-get install -y nginx` ile NGINX’i kur,
    - Tercihen `/var/lib/apt/lists/*` içeriğini silerek apt cache’i temizle.

   Yani bu blok kavramsal olarak:

    - `apt-get update` çalıştırmalı,
    - `DEBIAN_FRONTEND=noninteractive apt-get install -y nginx` ile NGINX kurmalı,
    - Sonunda `rm -rf /var/lib/apt/lists/*` ile cache’i temizlemeli.

3. Kurduğun NGINX’in `PATH` içinde erişilebilir olduğundan emin ol. Böylece
   imajdan oluşturulan bir container içinde şu komutlar çalışabilmeli:

       nginx -v
       nginx -V

4. İmaj, NGINX’i başlatıp 80 numaralı porttan HTTP trafiği kabul edebilmeli.
   `Dockerfile` içindeki varsayılan `CMD` satırı bu amaçla hazırlanmıştır;
   senin asıl görevin NGINX’in doğru şekilde kurulmasını sağlamak.

---

## Başarı kriterleri

Bu egzersiz şu koşullar sağlandığında başarılı sayılır:

- Bu `Dockerfile` ile tanımlanan imaj, şu komutla sorunsuz build edilebiliyor:

      docker build -t <bir-isim> .

- İmaj içinde NGINX, işletim sistemi paket deposundan kurulu durumda
  (binary eksik değil, paket hatası yok).

- Bu imaja dayalı bir container içinde aşağıdaki komutlar hatasız çalışıyor:

      nginx -v
      nginx -V

- Container içinde NGINX başarılı şekilde başlatılabiliyor ve 80 numaralı portta
  dinleyerek `/` isteğine geçerli bir HTTP cevabı döndürebiliyor.

---

## İpuçları

- `apt-get install` sırasında etkileşimli soruları engellemek için
  `DEBIAN_FRONTEND=noninteractive` kullan:

      DEBIAN_FRONTEND=noninteractive apt-get install -y nginx

- Yeni bir imajda her zaman ilk önce:

      apt-get update

  çalıştır.

- İmaj boyutunu gereksiz büyütmemek için, `apt-get install` sonrası cache’i temizle:

      rm -rf /var/lib/apt/lists/*

- Build veya NGINX komutları hata verirse, çıkan mesajları dikkatlice oku;
  genellikle hangi paketin, hangi adımın eksik olduğunu doğrudan söyler.

