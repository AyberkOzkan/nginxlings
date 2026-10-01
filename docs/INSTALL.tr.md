# Nginxlings kurulumu

Bu rehber, Nginxlings'i bir Linux bilgisayara kurmak ve açmak için gereken her
şeyi anlatır. Komut satırı bilgisi gerekmez; gereken birkaç komut olduğu gibi
kopyalanabilir.

## 1. Dosyayı indir

Eğitmeninin verdiği bağlantıdan ya da projenin GitHub sayfasındaki son sürümden
(release) kendi dağıtımına uygun dosyayı indir:

| Dağıtımın | İndireceğin dosya |
| --- | --- |
| Ubuntu, Debian, Linux Mint, Pop!_OS | `nginxlings_<sürüm>_amd64.deb` |
| Fedora, Rocky, AlmaLinux, RHEL, openSUSE | `nginxlings-<sürüm>-1.x86_64.rpm` |
| Arch, Manjaro | `nginxlings-<sürüm>-1-x86_64.pkg.tar.zst` |
| Diğerleri (Alpine, Void ...) | `nginxlings_<sürüm>_linux_amd64.tar.gz` |

Hangi dağıtımı kullandığını bilmiyorsan, bir terminal aç ve `cat /etc/os-release`
yaz; `NAME=` satırı dağıtımın adıdır. Bilgisayarın ARM işlemciliyse (ör. bazı
dizüstüler), adında `arm64` ya da `aarch64` geçen dosyayı indir.

## 2. Kur

İndirdiğin dosyaya çift tıkla. Uygulama merkezi (software center) açılır,
"Kur" düğmesine bas ve şifreni gir. Çift tıklama işe yaramazsa, dosyanın
bulunduğu klasörde bir terminal aç ve dağıtımına uyan komutu yaz:

```sh
sudo apt install ./nginxlings_*_amd64.deb                     # Ubuntu, Debian, Mint
sudo dnf install ./nginxlings-*.x86_64.rpm                    # Fedora, Rocky, Alma
sudo zypper install --allow-unsigned-rpm ./nginxlings-*.rpm   # openSUSE
sudo pacman -U ./nginxlings-*.pkg.tar.zst                     # Arch, Manjaro
```

Diğer dağıtımlarda `.tar.gz` dosyasını çift tıklayıp bir klasöre çıkar, o
klasörde bir terminal aç ve `./install.sh` yaz.

Kurulum, egzersizlerin ihtiyaç duyduğu her şeyi de kurar: NGINX, curl, Python ve
Docker. Docker'ı da başlatır.

## 3. Aç

Uygulama menüsünde "Nginxlings"i bul ve aç ya da bir terminal açıp
`nginxlings` yaz. Program ilk açılışta egzersizleri ev klasöründeki
`nginxlings` klasörüne koyar. Egzersiz dosyalarını orada düzenlersin.
Sağ paneldeki "Dosyalar:" satırı seçili egzersizin klasörünü gösterir, `o` tuşu
da o klasörü dosya yöneticisinde açar.

Temel tuşlar:

- `W` / `S`: egzersiz seç
- `c`: çözümünü kontrol et
- `h`: ipucu
- `Tab`: talimatlarla son kontrol çıktısı arasında geç
- `?`: bütün tuşlar
- `q`: çık

## 4. Altta "! Kurulum" yazıyorsa

Bir şey eksikse durum satırında `! Kurulum` uyarısı görünür. `d` tuşuna bas:
Nginxlings neyin eksik olduğunu ve nasıl düzeleceğini yazar. `f` tuşu
gösterilen komutu senin için çalıştırır ve terminalde şifreni sorar.

En sık görülen durum, kullanıcının Docker'ı kullanma izninin olmamasıdır. `f`
seni `docker` grubuna ekler. Bundan sonra **oturumu kapatıp yeniden açman** (ya
da bilgisayarı yeniden başlatman) gerekir; ardından Nginxlings'i yeniden aç.

## Rocky, AlmaLinux ve RHEL hakkında

Bu dağıtımlarda kurulum Docker yerine Podman'ın `docker` komutunu kurar.
Egzersiz 10 dışındaki bütün egzersizler onunla çalışır. Egzersiz 10, Docker'ın
`127.0.0.11` adresindeki DNS sunucusunu kullanır. O egzersiz için eğitmeninden
Docker CE kurulumunda yardım iste.

## Kaldırma

Paketi kaldırmak için `sudo apt remove nginxlings`, `sudo dnf remove nginxlings`
ya da `sudo pacman -R nginxlings` yaz; `install.sh` ile kurduysan
`./install.sh --uninstall` çalıştır. Ev klasöründeki `nginxlings` klasörü, yani
senin çalışmaların, silinmez.
